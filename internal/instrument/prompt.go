package instrument

import "bytes"

// PromptV1 is the v0.1 measurement prompt: it instructs the model to
// convert a specification into Go type definitions only (no
// implementation logic), matching the protocol used in
// docs/investigation/_sanity/gen.sh. Named and versioned (V1) because
// this string is instrument-relative config (REQ-MSR-04) — changing
// its wording changes what is being measured, so a future revision
// must be a new named constant (PromptV2), never a silent edit to this
// one, or historical reports become unreproducible.
const PromptV1 = "You convert software specifications into Go type definitions. Output ONLY one ```go code block containing: package declaration, type definitions (structs, named types, consts) and function signatures with empty bodies {}. No explanations, no comments, no implementation logic."

// BuildPrompt concatenates PromptV1 with the spec content into the single
// user-role message sent to the instrument. This is a v0.1 simplification:
// the reference experiment (gen.sh) used a system+user message split, but
// Generator takes one prompt string and Ollama.Generate sends only a
// single user-role message — instrument-relativity is about fixing *some*
// prompt and reporting it, not matching gen.sh's message structure exactly.
func BuildPrompt(specContent []byte) string {
	return PromptV1 + "\n\n" + string(specContent)
}

var (
	goFenceOpen = []byte("```go")
	fenceClose  = []byte("```")
)

// ExtractGoBlock scans text for the first fenced code block opened by a
// line that is exactly "```go" (lowercase, no leading whitespace, nothing
// else on the line) and closed by the next line that is exactly "```" (no
// leading whitespace, nothing else). It returns the bytes strictly between
// those two fence lines and true.
//
// Only the first such block is returned, even if the text contains
// several — this intentionally diverges from gen.sh's awk script, which
// concatenates every ```go block it finds.
//
// If no line exactly matches the opening fence, ExtractGoBlock returns
// (nil, false). If an opening fence is found but no matching closing fence
// follows before EOF, it also returns (nil, false) — an unterminated block
// is treated as "no extraction", not as "extract to EOF", so a truncated
// generation (e.g. cut off by num_predict) is never silently accepted as
// valid Go source.
//
// ```go, ```python and other language-tagged or indented fence variants
// are intentionally not matched: this mirrors gen.sh's exact behavior, not
// a broader Markdown fence parser.
func ExtractGoBlock(text []byte) ([]byte, bool) {
	// Normalize CRLF to LF first: a model or proxy that emits \r\n would
	// otherwise leave every line with a trailing \r, so the exact-match
	// fence comparisons below would never succeed — every generation
	// would look like "no code block found" and inflate the discard rate,
	// masking a formatting artifact as measurement noise.
	text = bytes.ReplaceAll(text, []byte("\r\n"), []byte("\n"))
	lines := bytes.Split(text, []byte("\n"))

	open := -1
	for i, line := range lines {
		if bytes.Equal(line, goFenceOpen) {
			open = i
			break
		}
	}
	if open == -1 {
		return nil, false
	}

	for j := open + 1; j < len(lines); j++ {
		if bytes.Equal(lines[j], fenceClose) {
			if j == open+1 {
				return []byte{}, true
			}
			return bytes.Join(lines[open+1:j], []byte("\n")), true
		}
	}
	return nil, false
}
