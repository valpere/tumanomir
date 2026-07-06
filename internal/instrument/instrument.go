// Package instrument provides the pluggable generation interface for the
// stochastic measurement layer (REQ-MSR-03). This is the only package
// allowed to make network calls; internal/metrics and internal/spec must
// stay network-free (REQ-CHK-05).
package instrument

import "context"

// DoneReasonLength is the backend-reported stop reason indicating output
// was cut off by the generation length limit (num_predict for Ollama) —
// the direct truncation signal Generation.DoneReason carries. Exported as
// a named constant, not a string literal at call sites, so a typo can't
// silently defeat the REQ-MSR-06 truncation check.
const DoneReasonLength = "length"

// DoneReasonStop is the backend-reported stop reason for a generation that
// completed naturally (the model emitted its own stop token/sequence).
const DoneReasonStop = "stop"

// Generation is one instrument response, carrying both the generated text
// and the backend's own token-count telemetry. PromptEvalCount and
// EvalCount are ground truth for detecting the truncation failure modes
// REQ-MSR-06 exists to prevent — e.g. EvalCount == NumPredict strongly
// suggests the output was cut off rather than stopping naturally.
// DoneReason ("stop" vs "length" for Ollama) is a stronger, direct signal
// than inferring truncation from EvalCount == NumPredict — prefer it when
// available and non-empty.
type Generation struct {
	// Text is the raw generated content, exactly as returned by the
	// backend — extraction of the actual Go source from any surrounding
	// prose/fencing happens downstream (see ExtractGoBlock in prompt.go).
	Text []byte
	// PromptEvalCount is how many tokens the backend itself counted in the
	// prompt — ground truth for cross-checking InstrumentConfig.NumCtx
	// against the actual prompt size (see EstimatePromptTokens's
	// under-estimate detection in prompt.go).
	PromptEvalCount int
	// EvalCount is how many tokens the backend generated for this
	// response — compared against InstrumentConfig.NumPredict to help
	// infer truncation when DoneReason is unavailable or ambiguous.
	EvalCount int
	// DoneReason is the backend-reported stop reason — "stop" (natural
	// completion) or "length" (cut off by NumPredict) for Ollama, see
	// DoneReasonStop/DoneReasonLength above. May be empty for backends
	// that don't report it, in which case callers fall back to comparing
	// EvalCount against NumPredict.
	DoneReason string
}

// Generator is the pluggable instrument interface. v0.1 ships one backend,
// Ollama (see ollama.go); implementations must honor the instrument
// configuration's think/num_ctx/num_predict fields exactly — these are
// measurement-integrity requirements, not defaults to optimize away.
type Generator interface {
	// Generate produces one generation for prompt, honoring ctx
	// cancellation. Implementations own their own retry/timeout policy for
	// transient backend errors; a returned error is treated by callers
	// (runMeasureWithGenerator in cmd/tumanomir) as a hard failure of the
	// whole measurement run, not a per-sample condition to retry around —
	// see that function's error-signaling contract.
	Generate(ctx context.Context, prompt string) (Generation, error)
}
