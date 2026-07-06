package instrument

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valpere/tumanomir/internal"
)

// defaultBaseURL is used when Ollama.BaseURL is the zero value.
const defaultBaseURL = "http://localhost:11434"

// defaultTimeout is used when Ollama.Timeout is the zero value. It's a
// generous safety bound against a truly hung connection, not a tight SLA —
// local models on modest hardware can be slow.
const defaultTimeout = 5 * time.Minute

// Ollama is the v0.1 Generator backend, talking to Ollama's /api/chat
// endpoint (stream:false, one complete JSON object per request).
//
// BaseURL defaults to defaultBaseURL ("http://localhost:11434") when empty.
// HTTPClient defaults to a client with a defaultTimeout timeout when nil.
// Use NewOllama for a constructed instance with Config set, or build the
// struct directly and rely on the zero-value defaults for
// BaseURL/HTTPClient/Timeout.
type Ollama struct {
	// BaseURL is the Ollama server root, e.g. "http://localhost:11434" or a
	// cloud endpoint. Empty means defaultBaseURL.
	BaseURL string

	// Timeout bounds each HTTP request to Ollama's /api/chat. Zero means
	// defaultTimeout (5 minutes) is used. Ignored if HTTPClient is set
	// explicitly — the caller owns that client's timeout behavior.
	Timeout time.Duration

	// HTTPClient, if set, is used as-is (Timeout above is then ignored).
	// Tests inject a client pointed at an httptest.Server here.
	HTTPClient *http.Client
	// Config is the full instrument configuration this backend generates
	// under — model, temperature, think/num_ctx/num_predict, etc.
	Config internal.InstrumentConfig
}

// NewOllama returns an Ollama backend for the given instrument
// configuration, using the default BaseURL and HTTP client.
func NewOllama(config internal.InstrumentConfig) *Ollama {
	return &Ollama{Config: config}
}

// chatRequest is the /api/chat request payload. Stream is always false —
// this project reads one complete JSON response object per request, never
// Ollama's streaming NDJSON mode.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Think    bool          `json:"think"`
	Options  chatOptions   `json:"options"`
}

// chatMessage is one entry in a chat request/response's message list.
// Generate always sends exactly one, Role "user" — no system prompt or
// multi-turn history, since each measurement sample is an independent
// single-shot generation.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatOptions carries the per-request generation parameters Ollama reads
// from InstrumentConfig — the measurement-integrity-critical fields
// (REQ-MSR-06) that must reach the backend exactly as configured.
type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
	NumPredict  int     `json:"num_predict"`
}

// chatResponse is the /api/chat response payload. Error is populated on
// failure responses; Ollama may also signal failure via a non-2xx status.
type chatResponse struct {
	Model           string      `json:"model"`
	Message         chatMessage `json:"message"`
	Done            bool        `json:"done"`
	DoneReason      string      `json:"done_reason"`
	PromptEvalCount int         `json:"prompt_eval_count"`
	EvalCount       int         `json:"eval_count"`
	Error           string      `json:"error"`
}

// EstimatePromptTokens is the conservative, stdlib-only token estimate
// used by Generate's preflight (no tokenizer in v0.1): ~3 bytes/token,
// rounded up, errs toward refusing rather than passing a prompt that
// would truncate the output. Plain len(prompt)/3 would round down and
// could underestimate, defeating the "errs toward refusing" guarantee.
//
// The guarantee itself only holds for ~ASCII input: Cyrillic UTF-8 runs
// ~2 bytes/char, so this heuristic can still under-count real token usage
// for non-ASCII specs (this project's own specs are Ukrainian). Exported
// so callers can independently cross-check a completed generation's
// actual PromptEvalCount against the estimate that gated it, since a
// stdlib-only pre-flight fix isn't possible without a real tokenizer.
func EstimatePromptTokens(prompt string) int {
	return (len(prompt) + 2) / 3
}

// Generate implements Generator. It runs a stdlib-only prompt-size
// preflight before making any HTTP call (REQ-MSR-06): num_ctx must have
// headroom for both the prompt and the requested output length, or the
// generation would be silently truncated.
func (o *Ollama) Generate(ctx context.Context, prompt string) (Generation, error) {
	cfg := o.Config

	estimatedPromptTokens := EstimatePromptTokens(prompt)
	if estimatedPromptTokens+cfg.NumPredict > cfg.NumCtx {
		return Generation{}, fmt.Errorf(
			"instrument: estimated prompt tokens (%d, len(prompt)/3 heuristic) + num_predict (%d) exceeds num_ctx (%d); increase num_ctx or reduce num_predict",
			estimatedPromptTokens, cfg.NumPredict, cfg.NumCtx)
	}

	// Build the request body from cfg exactly — no field here is optional
	// or "tunable for convenience"; Think/NumCtx/NumPredict in particular
	// are measurement-integrity requirements, not knobs.
	reqBody := chatRequest{
		Model:    cfg.Model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
		Think:    cfg.Think,
		Options: chatOptions{
			Temperature: cfg.Temperature,
			NumCtx:      cfg.NumCtx,
			NumPredict:  cfg.NumPredict,
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return Generation{}, fmt.Errorf("instrument: marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL()+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return Generation{}, fmt.Errorf("instrument: build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient().Do(httpReq)
	if err != nil {
		return Generation{}, fmt.Errorf("instrument: ollama request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the full body before decoding rather than streaming through
	// json.Decoder directly off resp.Body — stream:false means Ollama
	// sends exactly one complete JSON object, so there's no benefit to
	// incremental decoding, and reading fully first lets a decode error
	// still report the HTTP status code alongside it (see below).
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Generation{}, fmt.Errorf("instrument: read ollama response: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return Generation{}, fmt.Errorf("instrument: decode ollama response (status %d): %w", resp.StatusCode, err)
	}

	// Ollama can signal failure two ways: a non-2xx HTTP status, or (less
	// commonly) a 200 response whose body still carries a populated
	// Error field — check both rather than trusting the status code alone.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || chatResp.Error != "" {
		if isUnsupportedThinkError(chatResp.Error) {
			// A distinct, actionable error for the specific case of
			// requesting think mode on a model that doesn't support it —
			// worth surfacing separately since the fix (Think=false) is
			// immediate and specific, unlike a generic API error.
			return Generation{}, fmt.Errorf(
				"instrument: model %q does not support think mode; set InstrumentConfig.Think=false for this model (ollama: %s)",
				cfg.Model, chatResp.Error)
		}
		return Generation{}, fmt.Errorf("instrument: ollama API error (status %d): %s", resp.StatusCode, chatResp.Error)
	}

	return Generation{
		Text:            []byte(chatResp.Message.Content),
		PromptEvalCount: chatResp.PromptEvalCount,
		EvalCount:       chatResp.EvalCount,
		DoneReason:      chatResp.DoneReason,
	}, nil
}

// isUnsupportedThinkError reports whether msg is Ollama's "model does not
// support thinking" error. The exact wording is a moving target across
// Ollama versions, so a case-insensitive substring check on the two
// distinguishing words is sufficient for v0.1.
func isUnsupportedThinkError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "does not support") && strings.Contains(lower, "think")
}

// baseURL returns o.BaseURL with any trailing slash trimmed (so callers can
// safely append "/api/chat" without producing a double slash), or
// defaultBaseURL if unset.
func (o *Ollama) baseURL() string {
	if o.BaseURL != "" {
		return strings.TrimRight(o.BaseURL, "/")
	}
	return defaultBaseURL
}

// httpClient returns o.HTTPClient as-is if the caller supplied one
// (tests do, pointed at an httptest.Server), otherwise constructs a fresh
// client using o.timeout().
func (o *Ollama) httpClient() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: o.timeout()}
}

// timeout returns o.Timeout if non-zero, else defaultTimeout.
func (o *Ollama) timeout() time.Duration {
	if o.Timeout != 0 {
		return o.Timeout
	}
	return defaultTimeout
}
