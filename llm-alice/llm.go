package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	configPath = "/data/llm/llm-alice.json"
	keyPath    = "/data/llm/key"
)

type Config struct {
	BaseURL        string   `json:"base_url"`
	Model          string   `json:"model"`
	FallbackModel  string   `json:"fallback_model"`
	SystemPrompt   string   `json:"system_prompt"`
	Triggers       []string `json:"triggers"`
	MaxChars       int      `json:"max_chars"`
	MaxTokens      int      `json:"max_tokens"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	HistoryTurns   int      `json:"history_turns"`
	HistoryMinutes int      `json:"history_minutes"`
	Retries        int      `json:"retries"`

	// Routing. Rules: phrases starting with these go straight to the model.
	Rules []string `json:"rules"`
	// Alice's answers that mean she cannot help: the question then goes to
	// the model instead, before she speaks.
	HelplessAnswers []string `json:"helpless_answers"`
	// Answers that only quote a website, handled like helpless ones while
	// WebAnswersToLLM is on.
	WebAnswerMarkers []string `json:"web_answer_markers"`
	WebAnswersToLLM  bool     `json:"web_answers_to_llm"`
	// Phrases that forget the conversation.
	ResetPhrases []string `json:"reset_phrases"`
	// Phrases that always belong to Alice, even in the middle of a
	// conversation with the model: with the microphone already open the wake
	// word is often not recognised, so they cannot be told apart otherwise.
	AlicePhrases []string       `json:"alice_phrases"`
	ExtraParams  map[string]any `json:"extra_params"`
}

func defaultConfig() Config {
	return Config{
		BaseURL:       "https://generativelanguage.googleapis.com/v1beta/openai",
		Model:         "models/gemini-flash-lite-latest",
		FallbackModel: "models/gemini-flash-latest",
		SystemPrompt: "Ты — Gemini, языковая модель Google. С тобой говорят голосом " +
			"через умную колонку: вопрос распознаёт Яндекс Алиса, а твой ответ она " +
			"читает вслух своим голосом. Отвечай по-русски, коротко и по делу: не больше " +
			"трёх-четырёх предложений. Пиши простым текстом без списков, разметки, " +
			"эмодзи и ссылок. Если тему есть куда развить и собеседник явно " +
			"заинтересован, закончи ответ одним коротким встречным вопросом. " +
			"Если вопрос простой и ответ исчерпывающий, вопрос в конце не нужен.",
		Triggers: []string{
			"спроси у нейросети", "спроси нейросеть", "спроси нейросети",
			"спроси у гемини", "спроси гемини", "спроси джемини", "спроси gemini",
			"спроси у чат", "спроси чат",
		},
		MaxChars:       0,
		Retries:        2,
		MaxTokens:      400,
		TimeoutSeconds: 20,
		HistoryTurns:   4,
		Rules: []string{
			"объясни", "сравни", "почему", "как работает", "в чем разница",
			"придумай", "сочини",
		},
		HelplessAnswers: []string{
			"ничего не нашла", "ничего не нашлось", "не получилось ничего найти",
			"ничего найти не получилось", "не удалось найти", "пока не умею",
			"не знаю", "не поняла", "что-то пошло не так", "нет ответа",
			"не могу ответить", "в двух словах не рассказать", "быстро не разобраться",
			"сложный вопрос", "так просто не объяснить", "результатами поиска",
		},
		WebAnswerMarkers: []string{
			"нашла ответ на", "дает такой ответ", "ответ есть на", "нашла на сайте",
			"вот что нашла", "вот что пишут", "нашлось на",
		},
		WebAnswersToLLM: true,
		AlicePhrases: []string{
			"который час", "сколько времени", "какое сегодня число", "какой сегодня день",
			"какая погода", "какая сегодня погода", "погода", "поставь таймер", "таймер",
			"поставь будильник", "разбуди меня", "будильник", "напомни",
		},
		ResetPhrases: []string{
			"начни новый разговор", "начать новый разговор", "забудь разговор",
			"забудь наш разговор", "новая тема",
		},
		HistoryMinutes: 5,
	}
}

// loadConfig reads the optional JSON config over the defaults, and the key.
func loadConfig() (Config, string, error) {
	c := defaultConfig()
	if raw, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, "", fmt.Errorf("%s: %w", configPath, err)
		}
	}
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		return c, "", fmt.Errorf("no API key in %s", keyPath)
	}
	key := strings.TrimSpace(string(raw))
	if key == "" {
		return c, "", fmt.Errorf("empty API key in %s", keyPath)
	}
	return c, key, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	cfg  Config
	key  string
	http *http.Client

	mu       sync.Mutex
	history  []message
	lastAt   time.Time
	lastUsed atomic.Value // model that produced the last answer
}

// Reset forgets the conversation.
func (c *Client) Reset() {
	c.mu.Lock()
	c.history = nil
	c.mu.Unlock()
}

// LastModel reports which model answered last, for the journal.
func (c *Client) LastModel() string {
	s, _ := c.lastUsed.Load().(string)
	return s
}

func newClient(cfg Config, key string) *Client {
	return &Client{cfg: cfg, key: key, http: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second}}
}

// Ask sends the question with the recent conversation and returns the
// answer, cleaned up for speech.
func (c *Client) Ask(ctx context.Context, question string) (string, error) {
	c.mu.Lock()
	if time.Since(c.lastAt) > time.Duration(c.cfg.HistoryMinutes)*time.Minute {
		c.history = nil
	}
	msgs := append([]message{{"system", c.cfg.SystemPrompt}}, c.history...)
	msgs = append(msgs, message{"user", question})
	c.mu.Unlock()

	model := c.cfg.Model
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			// Overloaded model: back off, and try the fallback once.
			time.Sleep(time.Duration(attempt) * 1500 * time.Millisecond)
			if c.cfg.FallbackModel != "" && attempt == c.cfg.Retries {
				model = c.cfg.FallbackModel
			}
			logf("retry %d with model %s after: %v", attempt, model, lastErr)
		}
		answer, err := c.ask(ctx, msgs, model)
		if err == nil {
			c.lastUsed.Store(model)
			c.remember(question, answer)
			return answer, nil
		}
		lastErr = err
		if !retriable(err) || ctx.Err() != nil {
			return "", err
		}
	}
	return "", lastErr
}

// retriable reports whether another attempt makes sense: the model being busy
// or a transport hiccup, rather than a bad request or a wrong key.
func retriable(err error) bool {
	s := err.Error()
	for _, k := range []string{"HTTP 429", "HTTP 500", "HTTP 502", "HTTP 503", "HTTP 504",
		"timeout", "connection", "EOF", "temporarily"} {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func (c *Client) remember(question, answer string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.history = append(c.history, message{"user", question}, message{"assistant", answer})
	if n := 2 * c.cfg.HistoryTurns; len(c.history) > n {
		c.history = c.history[len(c.history)-n:]
	}
	c.lastAt = time.Now()
}

func (c *Client) ask(ctx context.Context, msgs []message, model string) (string, error) {
	body := map[string]any{"model": model, "messages": msgs, "max_tokens": c.cfg.MaxTokens}
	for k, v := range c.cfg.ExtraParams {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(c.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, shorten(string(data), 300))
	}
	var out struct {
		Choices []struct {
			Message message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("bad response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("empty response")
	}
	answer := forSpeech(out.Choices[0].Message.Content, c.cfg.MaxChars)
	if answer == "" {
		return "", errors.New("empty answer")
	}
	return answer, nil
}

var (
	markup = regexp.MustCompile("[*_#`>~|]+")
	spaces = regexp.MustCompile(`\s+`)
	links  = regexp.MustCompile(`https?://\S+`)
)

// forSpeech strips markup and cuts the text at a sentence boundary so that it
// is no longer than max characters.
func forSpeech(s string, max int) string {
	s = links.ReplaceAllString(s, "")
	s = markup.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\n- ", ". ")
	s = strings.TrimSpace(spaces.ReplaceAllString(s, " "))
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)[:max]
	cut := string(r)
	if i := strings.LastIndexAny(cut, ".!?"); i > len(cut)/2 {
		return cut[:i+1]
	}
	if i := strings.LastIndex(cut, " "); i > 0 {
		return cut[:i] + "…"
	}
	return cut
}

func shorten(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
