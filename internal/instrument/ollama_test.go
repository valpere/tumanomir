package instrument

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valpere/tumanomir/internal"
)

func baseConfig() internal.InstrumentConfig {
	return internal.InstrumentConfig{
		Backend:     "ollama",
		Model:       "qwen3-coder:30b",
		Temperature: 1.0,
		Samples:     10,
		Think:       false,
		NumCtx:      8192,
		NumPredict:  2048,
	}
}

func TestOllamaGenerateThinkFalseInPayload(t *testing.T) {
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Message: chatMessage{Role: "assistant", Content: "package main"},
			Done:    true,
		})
	}))
	defer srv.Close()

	cfg := baseConfig()
	cfg.Think = false
	o := &Ollama{BaseURL: srv.URL, Config: cfg}
	if _, err := o.Generate(context.Background(), "generate a Go file"); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if gotReq.Think {
		t.Fatalf("want think=false in request, got think=true")
	}
	if gotReq.Model != cfg.Model {
		t.Fatalf("want model %q, got %q", cfg.Model, gotReq.Model)
	}
	if gotReq.Stream {
		t.Fatalf("want stream=false, got stream=true")
	}
	if gotReq.Options.NumCtx != cfg.NumCtx || gotReq.Options.NumPredict != cfg.NumPredict {
		t.Fatalf("want num_ctx=%d num_predict=%d, got %+v", cfg.NumCtx, cfg.NumPredict, gotReq.Options)
	}
}

func TestOllamaGenerateThinkTrueInPayload(t *testing.T) {
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(chatResponse{
			Message: chatMessage{Role: "assistant", Content: "package main"},
			Done:    true,
		})
	}))
	defer srv.Close()

	cfg := baseConfig()
	cfg.Think = true
	o := &Ollama{BaseURL: srv.URL, Config: cfg}
	if _, err := o.Generate(context.Background(), "generate a Go file"); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !gotReq.Think {
		t.Fatalf("want think=true in request, got think=false")
	}
}

func TestOllamaGenerateParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{
			Message:         chatMessage{Role: "assistant", Content: "package main\n\nfunc main() {}\n"},
			Done:            true,
			DoneReason:      "stop",
			PromptEvalCount: 450,
			EvalCount:       890,
		})
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig()}
	got, err := o.Generate(context.Background(), "generate a Go file")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(got.Text) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("unexpected text: %q", got.Text)
	}
	if got.PromptEvalCount != 450 || got.EvalCount != 890 {
		t.Fatalf("want PromptEvalCount=450 EvalCount=890, got %+v", got)
	}
}

func TestOllamaGeneratePreflightRejectsWithoutHTTPCall(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	cfg := baseConfig()
	cfg.NumCtx = 100
	cfg.NumPredict = 50
	// len(prompt)/3 + NumPredict must exceed NumCtx to trigger rejection.
	prompt := strings.Repeat("x", 300) // estimate: 100 tokens; 100+50 > 100
	o := &Ollama{BaseURL: srv.URL, Config: cfg}

	_, err := o.Generate(context.Background(), prompt)
	if err == nil {
		t.Fatal("want preflight error, got nil")
	}
	if called {
		t.Fatal("preflight must reject before making any HTTP call")
	}
}

func TestOllamaGenerateUnsupportedThink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(chatResponse{
			Error: `model "qwen2.5-coder:7b" does not support thinking`,
		})
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig()}
	_, err := o.Generate(context.Background(), "generate a Go file")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "does not support think mode") {
		t.Fatalf("want distinguishable unsupported-think error, got: %v", err)
	}
}

func TestOllamaGenerateGenericAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(chatResponse{
			Error: "internal server error",
		})
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig()}
	_, err := o.Generate(context.Background(), "generate a Go file")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if strings.Contains(err.Error(), "does not support think mode") {
		t.Fatalf("generic error must not be mistaken for the unsupported-think case: %v", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("want status code in error, got: %v", err)
	}
}

func TestNewOllamaDefaultsBaseURL(t *testing.T) {
	o := NewOllama(baseConfig())
	if o.baseURL() != defaultBaseURL {
		t.Fatalf("want default base URL %q, got %q", defaultBaseURL, o.baseURL())
	}
}

func TestOllamaGenerateTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(chatResponse{
			Message: chatMessage{Role: "assistant", Content: "package main"},
			Done:    true,
		})
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig(), Timeout: 50 * time.Millisecond}
	_, err := o.Generate(t.Context(), "generate a Go file")
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("want a net.Error with Timeout()==true (client-side deadline), got %v", err)
	}
}

func TestOllamaTimeoutZeroFallsBackToDefault(t *testing.T) {
	o := &Ollama{Config: baseConfig()}
	if got := o.timeout(); got != defaultTimeout {
		t.Fatalf("timeout() with zero-value Timeout = %v, want defaultTimeout (%v)", got, defaultTimeout)
	}
}

func TestOllamaHTTPClientNotOverriddenWhenCallerSupplied(t *testing.T) {
	callerClient := &http.Client{Timeout: 7 * time.Second}
	o := &Ollama{HTTPClient: callerClient, Timeout: 50 * time.Millisecond, Config: baseConfig()}
	if got := o.httpClient(); got != callerClient {
		t.Fatalf("want httpClient() to return the caller-supplied client unmodified, got %+v", got)
	}
}

// TestOllamaGenerateContextCancellation verifies that Generate aborts
// promptly on a cancelled context rather than waiting for the full
// round-trip, and that the resulting error chain still exposes
// context.Canceled through Generate's fmt.Errorf("...: %w", err) wrapping
// (Go's http.Client wraps context.Canceled in a *url.Error, and errors.Is
// unwraps through it via *url.Error's Unwrap method).
func TestOllamaGenerateContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(chatResponse{
			Message: chatMessage{Role: "assistant", Content: "package main"},
			Done:    true,
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(30*time.Millisecond, cancel)

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig()}
	start := time.Now()
	_, err := o.Generate(ctx, "generate a Go file")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("want error from cancelled context, got nil")
	}
	if elapsed >= 300*time.Millisecond {
		t.Fatalf("Generate did not return promptly after cancellation (took %v, server sleeps 500ms)", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want error chain to include context.Canceled, got: %v", err)
	}
}

// TestOllamaGenerateMalformedResponseBody verifies that a non-JSON response
// body (e.g. an HTML error page from a reverse proxy sitting in front of
// Ollama) produces a decode error that includes the HTTP status code, per
// Generate's fmt.Errorf("instrument: decode ollama response (status %d): %w", ...).
func TestOllamaGenerateMalformedResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Bad Gateway</body></html>"))
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Config: baseConfig()}
	_, err := o.Generate(context.Background(), "generate a Go file")
	if err == nil {
		t.Fatal("want error decoding a non-JSON response body, got nil")
	}
	if !strings.Contains(err.Error(), "decode ollama response") {
		t.Fatalf("want decode-error wording, got: %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("status %d", http.StatusOK)) {
		t.Fatalf("want HTTP status code %d in error, got: %v", http.StatusOK, err)
	}
}
