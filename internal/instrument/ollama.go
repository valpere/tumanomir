package instrument

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/valpere/tumanomir/internal"
)

// defaultBaseURL is used when Ollama.BaseURL is the zero value.
const defaultBaseURL = "http://localhost:11434"

// Ollama is the v0.1 Generator backend, talking to Ollama's /api/chat
// endpoint (stream:false, one complete JSON object per request).
//
// BaseURL defaults to defaultBaseURL ("http://localhost:11434") when empty.
// HTTPClient defaults to http.DefaultClient when nil. Use NewOllama for a
// constructed instance with Config set, or build the struct directly and
// rely on the zero-value defaults for BaseURL/HTTPClient.
type Ollama struct {
	BaseURL    string
	HTTPClient *http.Client
	Config     internal.InstrumentConfig
}

// NewOllama returns an Ollama backend for the given instrument
// configuration, using the default BaseURL and HTTP client.
func NewOllama(config internal.InstrumentConfig) *Ollama {
	return &Ollama{Config: config}
}

// chatRequest is the /api/chat request payload.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Think    bool          `json:"think"`
	Options  chatOptions   `json:"options"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

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

// Generate implements Generator. It runs a stdlib-only prompt-size
// preflight before making any HTTP call (REQ-MSR-06): num_ctx must have
// headroom for both the prompt and the requested output length, or the
// generation would be silently truncated.
func (o *Ollama) Generate(ctx context.Context, prompt string) (Generation, error) {
	cfg := o.Config

	// Conservative, stdlib-only token estimate (no tokenizer in v0.1):
	// ~3 bytes/token errs toward refusing rather than passing a prompt
	// that would truncate the output.
	estimatedPromptTokens := len(prompt) / 3
	if estimatedPromptTokens+cfg.NumPredict > cfg.NumCtx {
		return Generation{}, fmt.Errorf(
			"instrument: estimated prompt tokens (%d, len(prompt)/3 heuristic) + num_predict (%d) exceeds num_ctx (%d); increase num_ctx or reduce num_predict",
			estimatedPromptTokens, cfg.NumPredict, cfg.NumCtx)
	}

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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Generation{}, fmt.Errorf("instrument: read ollama response: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return Generation{}, fmt.Errorf("instrument: decode ollama response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || chatResp.Error != "" {
		if isUnsupportedThinkError(chatResp.Error) {
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

func (o *Ollama) baseURL() string {
	if o.BaseURL != "" {
		return o.BaseURL
	}
	return defaultBaseURL
}

func (o *Ollama) httpClient() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return http.DefaultClient
}
