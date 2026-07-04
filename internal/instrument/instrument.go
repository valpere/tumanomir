// Package instrument provides the pluggable generation interface for the
// stochastic measurement layer (REQ-MSR-03). This is the only package
// allowed to make network calls; internal/metrics and internal/spec must
// stay network-free (REQ-CHK-05).
package instrument

import "context"

// Generation is one instrument response, carrying both the generated text
// and the backend's own token-count telemetry. PromptEvalCount and
// EvalCount are ground truth for detecting the truncation failure modes
// REQ-MSR-06 exists to prevent — e.g. EvalCount == NumPredict strongly
// suggests the output was cut off rather than stopping naturally.
type Generation struct {
	Text            []byte // generated content
	PromptEvalCount int    // tokens the backend counted in the prompt
	EvalCount       int    // tokens the backend generated
}

// Generator is the pluggable instrument interface. v0.1 ships one backend,
// Ollama (see ollama.go); implementations must honor the instrument
// configuration's think/num_ctx/num_predict fields exactly — these are
// measurement-integrity requirements, not defaults to optimize away.
type Generator interface {
	Generate(ctx context.Context, prompt string) (Generation, error)
}
