// llm-alice lets Yandex Alice answer with an LLM on old quasar firmware
// (JBL Link Portable, quasar 3.21), without replacing her voice service.
//
// "Алиса, спроси нейросеть, <question>": the daemon reads the recognised text
// from aliced, drops Alice's own answer, mutes her for that moment, asks an
// OpenAI-compatible endpoint (Gemini by default) and has Alice speak the
// answer in her own voice through a server action (see speak).
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

const version = "0.4.4"

// ---- journal ----

const logPath = "/tmp/llm-alice.log"

var logMu sync.Mutex

func logf(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	if st, err := os.Stat(logPath); err == nil && st.Size() > 256<<10 {
		os.Remove(logPath)
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format("15:04:05.000")}, args...)...)
}

// ---- text ----

func normalize(s string) string {
	s = strings.ToLower(strings.ReplaceAll(s, "ё", "е"))
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// question returns the part of the phrase after a trigger, and whether a
// trigger was found at its start.
func question(text string, triggers []string) (string, bool) {
	n := normalize(text)
	for _, t := range triggers {
		t = normalize(t)
		if n == t {
			return "", true
		}
		if strings.HasPrefix(n, t+" ") {
			return strings.TrimSpace(n[len(t):]), true
		}
	}
	return "", false
}

// ---- shared state of the voice loop ----

var (
	aliceState atomic.Uint64
	stateAt    atomic.Int64 // mono() of the last state change
	spotterAt  atomic.Int64 // mono() of the last wake word
	ownSentAt  atomic.Int64 // mono() of our last request to Alice
	followUp   atomic.Int64 // mono(): a follow-up answer is expected

	// A user phrase whose answer from Alice is awaited: she stays muted until
	// it is classified as a proper answer or replaced by the model's.
	fallbackAt atomic.Int64

	// A silent "Слушаю." sent only to open the microphone; Alice stays muted
	// until the listening starts.
	silentOpenAt atomic.Int64

	// A dialog-scenario answer waiting to be confirmed by Alice's own frames.
	dialogText   atomic.Value // the answer as sent
	dialogSentAt atomic.Int64 // mono(), 0 when nothing is pending
)

// Timestamps are monotonic nanoseconds since the process started, never wall
// clock: the speaker's clock can jump by hours when it syncs, which would
// close every time window at once.
var processStart = time.Now()

func mono() int64 { return int64(time.Since(processStart)) + 1 }

func since(v *atomic.Int64) time.Duration {
	t := v.Load()
	if t == 0 {
		return time.Hour
	}
	return time.Duration(mono() - t)
}

// turn is one question being answered.
type turn struct {
	mu        sync.Mutex
	active    bool
	answering bool // our answer is on its way: stop dropping Alice's replies
	droppedAt time.Time
	startedAt time.Time
}

func (t *turn) get() (active, answering bool, droppedAt, startedAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active, t.answering, t.droppedAt, t.startedAt
}

var cur turn

// speak has Alice read the text. Answers that end in a question go through
// the dialog scenario, after which Alice keeps listening and the user can
// answer without the wake word. Everything else uses the smart-home phrase
// scenario, which is not filtered and leaves the microphone closed.
func speak(text string) {
	if endsWithQuestion(text) {
		now := mono()
		followUp.Store(now)
		dialogText.Store(text)
		dialogSentAt.Store(now)
		ownSentAt.Store(now)
		// Alice stays muted until her frames show our text: if the dialog
		// scenario refuses it, the refusal is never heard.
		if err := sayDialog(text); err != nil {
			logf("dialog say failed: %v", err)
			dialogSentAt.Store(0)
			speakPhrase(text)
			return
		}
		logf("say via dialog scenario (repeat_after_me, listens after): %q", text)
		if !reacted(now, 1500*time.Millisecond) {
			sayDialog(text)
			logf("not picked up, resent")
		}
		// Never stay muted if no frame with the text ever comes.
		go func(sent int64) {
			time.Sleep(5 * time.Second)
			if dialogSentAt.CompareAndSwap(sent, 0) {
				restoreAlice()
				logf("no echo of our text: unmuted anyway")
			}
		}(now)
		return
	}
	speakPhrase(text)
}

// speakPhrase reads the text through the smart-home phrase scenario.
func speakPhrase(text string) {
	restoreAlice()
	ownSentAt.Store(mono())
	sent := mono()
	if err := say(text); err != nil {
		logf("say failed: %v", err)
		return
	}
	logf("say via phrase scenario (quasar.iot.repeat_phrase, no listening): %q", text)
	// Right after a cancel aliced sometimes ignores one request.
	if !reacted(sent, 1500*time.Millisecond) {
		ownSentAt.Store(mono())
		say(text)
		logf("not picked up, resent")
	}
}

// echoes reports whether Alice's displayed text carries our answer (the
// dialog scenario adds a varying lead-in such as "Ваша фраза: «…»").
func echoes(speech, text string) bool {
	want := []rune(normalize(text))
	if len(want) > 30 {
		want = want[:30]
	}
	return len(want) > 0 && strings.Contains(normalize(speech), string(want))
}

// stripWake removes a leading wake word from a recognised phrase. When the
// microphone is already open, "Алиса, …" arrives as part of the text and no
// spotter event is sent, yet it is still a new request. Only a leading one
// is removed, so that "расскажи про Алису в стране чудес" is left alone.
func stripWake(n string) (string, bool) {
	words := strings.Fields(n)
	if len(words) > 0 && (words[0] == "алиса" || words[0] == "алис") {
		return strings.Join(words[1:], " "), true
	}
	return n, false
}

// matchPrefix reports whether the normalised phrase starts with one of the
// given phrases as whole words.
func matchPrefix(n string, list []string) bool {
	for _, p := range list {
		p = normalize(p)
		if p != "" && (n == p || strings.HasPrefix(n, p+" ")) {
			return true
		}
	}
	return false
}

// helpless explains why Alice's answer should be replaced by the model's, or
// returns "" when she answered properly.
func helpless(speech string, cfg Config) string {
	n := normalize(speech)
	for _, k := range cfg.HelplessAnswers {
		if k = normalize(k); k != "" && strings.Contains(n, k) {
			return "cannot answer"
		}
	}
	if cfg.WebAnswersToLLM {
		for _, k := range cfg.WebAnswerMarkers {
			if k = normalize(k); k != "" && strings.Contains(n, k) {
				return "only quotes a website"
			}
		}
	}
	return ""
}

// fallbackCandidate tells whether Alice's answer to this phrase may be
// replaced by the model's. Playback and music phrases are left alone: their
// refusals belong to ytm-alice, which mutes Alice on its own.
func fallbackCandidate(n string) bool {
	if isCommand(n) {
		return false
	}
	for _, w := range []string{"ютуб", "youtube", "музык", "песн", "трек", "альбом", "плейлист", "радио"} {
		if strings.Contains(n, w) {
			return false
		}
	}
	return true
}

// isCommand recognises playback and volume commands, which belong to other
// handlers even right after a question from the model.
func isCommand(n string) bool {
	for _, w := range []string{"пауза", "стоп", "хватит", "продолжи", "дальше", "следующ",
		"назад", "предыдущ", "включи", "выключи", "громче", "тише", "что сейчас играет", "поставь"} {
		if strings.HasPrefix(n, w) {
			return true
		}
	}
	return false
}

func endsWithQuestion(s string) bool {
	s = strings.TrimRight(strings.TrimSpace(s), "\"»')")
	return strings.HasSuffix(s, "?")
}

func reacted(sent int64, max time.Duration) bool {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		if stateAt.Load() > sent {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return stateAt.Load() > sent
}

// waitSpoken waits until Alice has started speaking and then stopped. If she
// never starts within a few seconds, it returns anyway.
func waitSpoken(max time.Duration) {
	deadline := time.Now().Add(max)
	start := time.Now()
	started := false
	for time.Now().Before(deadline) {
		switch {
		case aliceState.Load() == 2:
			started = true
		case started:
			time.Sleep(300 * time.Millisecond) // let the tail finish
			return
		case time.Since(start) > 6*time.Second:
			return // she never started; do not wait for the full window
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// closeAutoListen ends the microphone session the firmware opens by itself
// after our requests, never one the user started with the wake word.
func closeAutoListen(spotterBefore int64) {
	for _, wait := range []time.Duration{time.Second, 1500 * time.Millisecond, 2 * time.Second} {
		time.Sleep(wait)
		if aliceState.Load() != 1 || spotterAt.Load() != spotterBefore {
			return
		}
		if since(&followUp) < 60*time.Second {
			return // the listening after a dialog answer is wanted
		}
		logf("auto-listen still open: cancel")
		cancelDialog()
	}
}

// answer replies to q with the model, or with a fixed text when given.
// alreadyDropped is set when Alice's own reply was cancelled before the call.
func answer(c *Client, q, fixed string, alreadyDropped bool) {
	cur.mu.Lock()
	cur.active, cur.answering, cur.droppedAt, cur.startedAt = true, false, time.Time{}, time.Now()
	if alreadyDropped {
		cur.droppedAt = time.Now()
	}
	cur.mu.Unlock()
	followUp.Store(0)
	muteAlice()
	defer func() {
		cur.mu.Lock()
		cur.active = false
		cur.mu.Unlock()
	}()

	var text string
	if fixed != "" {
		text = fixed
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.TimeoutSeconds)*time.Second)
		t0 := time.Now()
		a, err := c.Ask(ctx, q)
		cancel()
		if err != nil {
			logf("LLM error after %v: %v", time.Since(t0).Round(time.Millisecond), err)
			text = "Нейросеть сейчас не отвечает."
		} else {
			logf("LLM answered in %v by %s: %q", time.Since(t0).Round(time.Millisecond), c.LastModel(), a)
			text = a
		}
	}

	// Let Alice's own reply arrive and be dropped first: the cancel would
	// also cancel our answer if it were already in flight.
	for {
		_, _, dropped, started := cur.get()
		if (!dropped.IsZero() && time.Since(dropped) > 500*time.Millisecond) || time.Since(started) > 3*time.Second {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cur.mu.Lock()
	cur.answering = true
	cur.mu.Unlock()
	speak(text)
}

func serve() error {
	cfg, key, err := loadConfig()
	if err != nil {
		return err
	}
	c := newClient(cfg, key)
	restoreAlice() // a previous run may have died while Alice was muted
	logf("llm-alice %s started, model %s", version, cfg.Model)

	var (
		fallbackQ   string // the phrase whose answer from Alice is awaited
		lastQ       string // the last phrase Alice answered herself
		lastQAt     time.Time
		prev        uint64
		pendingText string
		pendingAt   time.Time
		last        string
		lastAt      time.Time
		lastMono    int64
	)
	for {
		conn, err := net.DialTimeout("tcp", ipcAddr, 2*time.Second)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		for {
			raw, err := readFrame(conn)
			if err != nil {
				break
			}
			ev := decode(raw)
			if ev.Spotted {
				spotterAt.Store(mono())
				// "Алиса, …" starts a new request, not an answer to the
				// model's question.
				followUp.Store(0)
			}
			if !ev.HasState {
				continue
			}
			if ev.State != prev {
				stateAt.Store(mono())
			}
			aliceState.Store(ev.State)

			// A dialog-scenario answer: unmute when Alice shows our text,
			// otherwise she refused it, so read it as a phrase instead.
			if sent := dialogSentAt.Load(); sent != 0 && (ev.State == 3 || ev.State == 2) && ev.Speech != "" {
				text, _ := dialogText.Load().(string)
				if dialogSentAt.CompareAndSwap(sent, 0) {
					if echoes(ev.Speech, text) {
						restoreAlice()
						logf("dialog answer accepted")
					} else {
						logf("dialog answer refused (%q): reading it as a phrase", ev.Speech)
						cancelDialog()
						go func() {
							time.Sleep(400 * time.Millisecond)
							speakPhrase(text)
							// The phrase scenario leaves the microphone closed.
							// A short neutral line through the dialog scenario
							// opens it, so the question can still be answered.
							waitSpoken(60 * time.Second)
							// Mute Alice for the word itself: only the listening
							// session it opens is wanted.
							muteAlice()
							now := mono()
							silentOpenAt.Store(now)
							followUp.Store(now)
							ownSentAt.Store(now)
							if err := sayDialog("Слушаю."); err != nil {
								silentOpenAt.Store(0)
								restoreAlice()
								return
							}
							logf("say via dialog scenario, muted: %q, to open listening after a refused dialog answer", "Слушаю.")
							time.Sleep(4 * time.Second)
							if silentOpenAt.CompareAndSwap(now, 0) {
								restoreAlice()
								logf("listening did not open: unmuted anyway")
							}
						}()
					}
				}
			}
			// Alice's own answer to a plain phrase: keep it if it is a real
			// answer, otherwise replace it with the model's before she speaks.
			if at := fallbackAt.Load(); at != 0 && (ev.State == 3 || ev.State == 2) && ev.Speech != "" {
				if active, _, _, _ := cur.get(); !active && since(&ownSentAt) > 5*time.Second &&
					fallbackAt.CompareAndSwap(at, 0) {
					if why := helpless(ev.Speech, cfg); why != "" {
						logf("Alice %s (%q): asking the model %q", why, ev.Speech, fallbackQ)
						cancelDialog()
						go answer(c, fallbackQ, "", true)
					} else {
						restore(false)
						logf("Alice answers %q herself: %q", fallbackQ, ev.Speech)
					}
				}
			}
			// While a question is being answered, drop whatever Alice wants
			// to say herself; it is visible in state 3 before she speaks.
			if active, answering, _, _ := cur.get(); active && !answering &&
				(ev.State == 3 || ev.State == 2) && ev.Speech != "" {
				logf("drop Alice's reply in state %d: %q", ev.State, ev.Speech)
				cancelDialog()
				cur.mu.Lock()
				cur.droppedAt = time.Now()
				cur.mu.Unlock()
			}
			// After our own request the firmware opens the microphone.
			if ev.State == 1 && prev != 1 && since(&spotterAt) > 2*time.Second && since(&ownSentAt) < 90*time.Second {
				if followUp.Load() != 0 && since(&followUp) < 90*time.Second {
					// The listening that follows a question: count the window
					// from now, not from when Alice started reading.
					followUp.Store(mono())
					if t := silentOpenAt.Load(); t != 0 && silentOpenAt.CompareAndSwap(t, 0) {
						restoreAlice()
					}
					logf("listening for a follow-up")
				} else if since(&ownSentAt) < 10*time.Second {
					go closeAutoListen(spotterAt.Load())
				}
			}
			prev = ev.State

			// quasar 3.21 sends the phrase in a state 1 frame and the final
			// state 3 in a separate one without text.
			text := ev.Text
			if text != "" {
				pendingText, pendingAt = text, time.Now()
			}
			if ev.State != 3 {
				continue
			}
			if text == "" && time.Since(pendingAt) < 10*time.Second {
				text = pendingText
			}
			pendingText = ""
			n, woke := stripWake(normalize(text))
			if woke {
				followUp.Store(0) // a new request, not an answer to the model
			}
			if n == "" {
				continue
			}
			// The same phrase without a wake word in between is one utterance
			// seen in several frames.
			if n == last && time.Since(lastAt) < 10*time.Second &&
				spotterAt.Load() <= lastMono && followUp.Load() <= lastMono {
				continue
			}
			last, lastAt, lastMono = n, time.Now(), mono()
			if active, _, _, _ := cur.get(); active {
				logf("busy, ignored %q", text)
				continue
			}
			switch q, trig := question(n, cfg.Triggers); {
			case matchPrefix(n, cfg.ResetPhrases):
				c.Reset()
				followUp.Store(0)
				logf("conversation reset by %q", n)
				go answer(c, "", "Начинаем новый разговор.", false)
			case trig && q == "" && lastQ != "" && time.Since(lastQAt) < 2*time.Minute:
				// "спроси нейросеть" right after Alice answered something:
				// ask the model the same question.
				logf("re-asking the model the last question %q", lastQ)
				go answer(c, lastQ, "", false)
			case trig && q == "":
				go answer(c, "", "Что спросить у нейросети?", false)
			case trig:
				logf("question %q", q)
				go answer(c, q, "", false)
			case since(&followUp) < 60*time.Second && !isCommand(n) && !matchPrefix(n, cfg.AlicePhrases):
				// A phrase right after an answer that ended in a question
				// continues that conversation.
				followUp.Store(0)
				logf("follow-up %q", n)
				go answer(c, n, "", false)
			case matchPrefix(n, cfg.Rules):
				logf("rule: question %q", n)
				go answer(c, n, "", false)
			default:
				// Alice answers this one herself. Keep her muted until her
				// answer shows up, so that a helpless one is never heard.
				lastQ, lastQAt = n, time.Now()
				if fallbackCandidate(n) {
					fallbackQ = n
					now := mono()
					fallbackAt.Store(now)
					mute(false)
					go func(at int64) {
						time.Sleep(3 * time.Second)
						if fallbackAt.CompareAndSwap(at, 0) {
							restore(false)
						}
					}(now)
				}
			}
		}
		conn.Close()
		time.Sleep(time.Second)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: llm-alice serve | ask [question] | version")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		if err := serve(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "ask":
		cfg, key, err := loadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		q := strings.Join(os.Args[2:], " ")
		if q == "" {
			q = "Ответь одним коротким предложением: кто ты?"
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
		defer cancel()
		t0 := time.Now()
		a, err := newClient(cfg, key).Ask(ctx, q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error after %v: %v\n", time.Since(t0).Round(time.Millisecond), err)
			os.Exit(1)
		}
		fmt.Printf("model %s, %v:\n%s\n", cfg.Model, time.Since(t0).Round(time.Millisecond), a)
	case "version":
		fmt.Println("llm-alice", version)
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}
