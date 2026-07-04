package instrument

import (
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	got := BuildPrompt([]byte("spec content here"))
	want := PromptV1 + "\n\nspec content here"
	if got != want {
		t.Fatalf("BuildPrompt() = %q, want %q", got, want)
	}
}

func TestExtractGoBlock(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		want   string
		wantOK bool
	}{
		{
			name:   "single well-formed block",
			text:   "some text\n```go\npackage main\n\nfunc main() {}\n```\ntrailing\n",
			want:   "package main\n\nfunc main() {}",
			wantOK: true,
		},
		{
			name:   "no fence present",
			text:   "package main\n\nfunc main() {}\n",
			want:   "",
			wantOK: false,
		},
		{
			name: "opening fence with no closing fence returns false",
			// Documented choice: an unterminated block is "no extraction",
			// not "extract to EOF" — a truncated generation must never be
			// silently accepted as valid Go source.
			text:   "```go\npackage main\n\nfunc main() {}\n",
			want:   "",
			wantOK: false,
		},
		{
			name:   "multiple go blocks: only the first is returned",
			text:   "```go\npackage first\n```\nsome prose in between\n```go\npackage second\n```\n",
			want:   "package first",
			wantOK: true,
		},
		{
			name:   "wrong language block is not matched",
			text:   "```python\nprint('hi')\n```\n",
			want:   "",
			wantOK: false,
		},
		{
			name:   "empty block: open immediately followed by close",
			text:   "```go\n```\n",
			want:   "",
			wantOK: true,
		},
		{
			name:   "CRLF line endings are normalized before fence matching",
			text:   "some text\r\n```go\r\npackage main\r\n\r\nfunc main() {}\r\n```\r\ntrailing\r\n",
			want:   "package main\n\nfunc main() {}",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractGoBlock([]byte(tt.text))
			if ok != tt.wantOK {
				t.Fatalf("ExtractGoBlock() ok = %v, want %v (got=%q)", ok, tt.wantOK, got)
			}
			if !tt.wantOK {
				if got != nil {
					t.Fatalf("ExtractGoBlock() want nil bytes on ok=false, got %q", got)
				}
				return
			}
			if string(got) != tt.want {
				t.Fatalf("ExtractGoBlock() = %q, want %q", got, tt.want)
			}
			if got == nil {
				t.Fatalf("ExtractGoBlock() ok=true but bytes is nil, want non-nil (even if empty) for %q", tt.name)
			}
		})
	}
}

func TestExtractGoBlockEmptyIsDistinctFromNotFound(t *testing.T) {
	// An empty block ([]byte{}, true) must be distinguishable from "no
	// extraction happened" (nil, false) at this layer — downstream code
	// (dispersion.ValidGo) is responsible for rejecting the empty content
	// as invalid Go, not this function conflating the two failure modes.
	got, ok := ExtractGoBlock([]byte("```go\n```\n"))
	if !ok {
		t.Fatalf("want ok=true for a found-but-empty block")
	}
	if len(got) != 0 {
		t.Fatalf("want empty bytes, got %q", got)
	}

	_, ok = ExtractGoBlock([]byte("no fence at all"))
	if ok {
		t.Fatalf("want ok=false when no fence is present")
	}
}
