package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestQuestion(t *testing.T) {
	trig := defaultConfig().Triggers
	cases := []struct {
		in string
		q  string
		ok bool
	}{
		{"Спроси нейросеть, сочини короткую сказку про робота", "сочини короткую сказку про робота", true},
		{"спроси у нейросети почему небо голубое?", "почему небо голубое", true},
		{"Спроси Gemini сколько лет Земле", "сколько лет земле", true},
		{"Спроси нейросеть", "", true},
		{"Включи на YouTube Daft Punk", "", false},
		{"Какая погода", "", false},
		{"спросить нейросеть", "", false},
	}
	for _, c := range cases {
		q, ok := question(c.in, trig)
		if q != c.q || ok != c.ok {
			t.Errorf("question(%q) = %q, %v; want %q, %v", c.in, q, ok, c.q, c.ok)
		}
	}
}

func TestForSpeech(t *testing.T) {
	in := "**Ответ:** небо голубое из-за\n- рассеяния Рэлея. Подробнее: https://example.com/x `код`"
	got := forSpeech(in, 0)
	if strings.ContainsAny(got, "*`\n") || strings.Contains(got, "http") {
		t.Fatalf("markup left: %q", got)
	}
	long := strings.Repeat("Это предложение. ", 100)
	cut := forSpeech(long, 120)
	if utf8.RuneCountInString(cut) > 120 || !strings.HasSuffix(cut, ".") {
		t.Fatalf("bad cut: %q", cut)
	}
}

func TestEndsWithQuestion(t *testing.T) {
	yes := []string{"Хотите пример?", "Так понятнее?", "«А что дальше?»", "Правда?\""}
	no := []string{"Это ответ.", "Вопрос был сложный", "", "Ответ! Да."}
	for _, s := range yes {
		if !endsWithQuestion(s) {
			t.Errorf("expected question: %q", s)
		}
	}
	for _, s := range no {
		if endsWithQuestion(s) {
			t.Errorf("unexpected question: %q", s)
		}
	}
}

func TestEchoes(t *testing.T) {
	ans := "Чёрные дыры — это области космоса с сильной гравитацией. Хотите узнать больше?"
	if !echoes("Ваша фраза: «"+ans+"»", ans) {
		t.Fatal("lead-in must not hide our text")
	}
	if echoes("Да вы оригинал.", ans) {
		t.Fatal("a refusal is not our text")
	}
}

func TestIsCommand(t *testing.T) {
	for _, s := range []string{"пауза", "дальше", "включи на ютубе daft punk", "громче"} {
		if !isCommand(s) {
			t.Errorf("expected command: %q", s)
		}
	}
	for _, s := range []string{"да хочу", "расскажи подробнее", "нет спасибо", "а почему"} {
		if isCommand(s) {
			t.Errorf("unexpected command: %q", s)
		}
	}
}

func TestRouting(t *testing.T) {
	cfg := defaultConfig()
	if !matchPrefix("объясни квантовую запутанность", cfg.Rules) || !matchPrefix("почему небо голубое", cfg.Rules) {
		t.Fatal("rules must match at the start")
	}
	if matchPrefix("какая погода", cfg.Rules) || matchPrefix("объяснительная записка", cfg.Rules) {
		t.Fatal("rules must match whole words only")
	}
	if !matchPrefix(normalize("Начни новый разговор"), cfg.ResetPhrases) || !matchPrefix("новая тема", cfg.ResetPhrases) {
		t.Fatal("reset phrases")
	}
	for _, n := range []string{"включи на ютубе daft punk", "пауза", "поставь музыку", "какая песня играет"} {
		if fallbackCandidate(n) {
			t.Errorf("must leave music phrases to ytm-alice: %q", n)
		}
	}
	for _, n := range []string{"который час", "какая погода завтра", "поставь таймер на пять минут"} {
		if !matchPrefix(n, cfg.AlicePhrases) {
			t.Errorf("must stay with Alice even mid-conversation: %q", n)
		}
	}
	if matchPrefix("да хочу", cfg.AlicePhrases) || matchPrefix("расскажи подробнее", cfg.AlicePhrases) {
		t.Fatal("follow-up answers must not look like Alice's phrases")
	}
	if !fallbackCandidate("какая высота эвереста") {
		t.Fatal("plain questions are candidates")
	}
}

func TestHelpless(t *testing.T) {
	cfg := defaultConfig()
	for _, s := range []string{
		"По вашему запросу я ничего не нашла.",
		"Ой, что-то пошло не так. Пожалуйста, попробуйте ещё раз.",
		"Я пока не умею это делать.",
		"Сайт nur.kz даёт такой ответ: Черные дыры - это…",
		"Нашла ответ на nur.kz. Слушайте: …",
		"К сожалению, не получилось ничего найти.",
		"По вашему запросу ничего не нашлось.",
		"Тут в двух словах не рассказать.",
		"Это сложный вопрос, быстро не разобраться.",
		"К сожалению, ничего найти не получилось.",
		"Так просто не объяснить.",
	} {
		if helpless(s, cfg) == "" {
			t.Errorf("expected helpless: %q", s)
		}
	}
	for _, s := range []string{"Сейчас в Москве плюс пять.", "Высота Эвереста 8848 метров.", "Поставила таймер на пять минут."} {
		if helpless(s, cfg) != "" {
			t.Errorf("a proper answer was replaced: %q", s)
		}
	}
	cfg.WebAnswersToLLM = false
	if helpless("Нашла ответ на nur.kz. Слушайте: …", cfg) != "" {
		t.Fatal("web answers must stay with Alice when the switch is off")
	}
}

func TestStripWake(t *testing.T) {
	cases := map[string]string{
		"алиса который час":                 "который час",
		"алиса спроси нейросеть":            "спроси нейросеть",
		"расскажи про алису в стране чудес": "расскажи про алису в стране чудес",
		"который час":                       "который час",
	}
	for in, want := range cases {
		if got, _ := stripWake(in); got != want {
			t.Errorf("stripWake(%q) = %q, want %q", in, got, want)
		}
	}
	if q, ok := question("спроси нейросеть", defaultConfig().Triggers); !ok || q != "" {
		t.Fatal("bare trigger after stripping the wake word")
	}
}

func TestDecode(t *testing.T) {
	spot := pb(43, pb(1, pb(1, []byte("алиса"))))
	if e := decode(spot); !e.Spotted || e.HasState {
		t.Fatalf("spotter: %+v", e)
	}
	listening := pb(3, append(pi(1, 1), pb(2, []byte("Спроси нейросеть"))...))
	if e := decode(listening); !e.HasState || e.State != 1 || e.Text != "Спроси нейросеть" {
		t.Fatalf("listening: %+v", e)
	}
	reply := pb(3, append(pi(1, 3), pb(3, pb(1, []byte("Я не знаю.")))...))
	if e := decode(reply); e.State != 3 || e.Speech != "Я не знаю." {
		t.Fatalf("reply: %+v", e)
	}
}

func TestAsk(t *testing.T) {
	var seen []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "bad request "+r.URL.Path, 400)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(raw, &body)
		seen = append(seen, body)
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"**Привет!** Я помощник."}}]}`)
	}))
	defer srv.Close()

	cfg := defaultConfig()
	cfg.BaseURL = srv.URL + "/v1/"
	cfg.ExtraParams = map[string]any{"temperature": 0.3, "max_tokens": nil}
	c := newClient(cfg, "k")
	a, err := c.Ask(context.Background(), "кто ты")
	if err != nil || a != "Привет! Я помощник." {
		t.Fatalf("ask: %q, %v", a, err)
	}
	if _, err := c.Ask(context.Background(), "а ещё"); err != nil {
		t.Fatal(err)
	}
	if seen[0]["temperature"] != 0.3 {
		t.Fatalf("extra param not sent: %v", seen[0])
	}
	if _, ok := seen[0]["max_tokens"]; ok {
		t.Fatal("null extra param must remove the field")
	}
	if msgs := seen[1]["messages"].([]any); len(msgs) != 4 {
		t.Fatalf("history not sent: %d messages", len(msgs))
	}

	bad := newClient(cfg, "wrong")
	if _, err := bad.Ask(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("expected HTTP 400, got %v", err)
	}
}

func TestMonotonicWindows(t *testing.T) {
	var v atomic.Int64
	if since(&v) < time.Minute {
		t.Fatal("an unset stamp must look old")
	}
	v.Store(mono())
	time.Sleep(20 * time.Millisecond)
	if d := since(&v); d < 15*time.Millisecond || d > time.Second {
		t.Fatalf("since = %v", d)
	}
	if a, b := mono(), mono(); b < a || a <= 0 {
		t.Fatalf("mono not increasing: %d %d", a, b)
	}
}
