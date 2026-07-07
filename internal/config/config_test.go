package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
)

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }
func ptrB(v bool) *bool       { return &v }
func ptrS(v string) *string   { return &v }

// --- Load: valid file, missing file, malformed YAML (⚖️ Balanced) ---

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tumanomir.yaml")
	content := `
thresholds:
  k_drift_max: 0.15
instrument:
  backend: ollama
  model: qwen3-coder:30b
  samples: 20
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Thresholds == nil || cfg.Thresholds.KDriftMax == nil || *cfg.Thresholds.KDriftMax != 0.15 {
		t.Fatalf("cfg.Thresholds = %+v, want KDriftMax=0.15", cfg.Thresholds)
	}
	if cfg.Instrument == nil || cfg.Instrument.Backend == nil || *cfg.Instrument.Backend != "ollama" {
		t.Fatalf("cfg.Instrument = %+v, want Backend=ollama", cfg.Instrument)
	}
	if cfg.Instrument.Samples == nil || *cfg.Instrument.Samples != 20 {
		t.Fatalf("cfg.Instrument.Samples = %v, want 20", cfg.Instrument.Samples)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("want an error for a missing config file, got nil")
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tumanomir.yaml")
	if err := os.WriteFile(path, []byte("thresholds: [this is not a mapping\n"), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("want an error for malformed YAML, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("want error to name the offending path %q, got %q", path, err.Error())
	}
}

// --- ApplyThresholds: every set/unset field combination (🏗️ Production) ---

func TestApplyThresholds(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		in   internal.Thresholds
		want internal.Thresholds
	}{
		{
			name: "nil Thresholds section leaves th untouched",
			cfg:  Config{},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
		},
		{
			name: "empty Thresholds section (all fields unset) leaves th untouched",
			cfg:  Config{Thresholds: &Thresholds{}},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
		},
		{
			name: "only KDriftMax set",
			cfg:  Config{Thresholds: &Thresholds{KDriftMax: ptrF(0.10)}},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.10, DConstMin: 0.35, DPairMax: 0.30},
		},
		{
			name: "only DConstMin set",
			cfg:  Config{Thresholds: &Thresholds{DConstMin: ptrF(0.50)}},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.50, DPairMax: 0.30},
		},
		{
			name: "only DPairMax set",
			cfg:  Config{Thresholds: &Thresholds{DPairMax: ptrF(0.40)}},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.40},
		},
		{
			name: "all three set, including an explicit zero",
			cfg: Config{Thresholds: &Thresholds{
				KDriftMax: ptrF(0.0),
				DConstMin: ptrF(0.60),
				DPairMax:  ptrF(0.05),
			}},
			in:   internal.Thresholds{KDriftMax: 0.20, DConstMin: 0.35, DPairMax: 0.30},
			want: internal.Thresholds{KDriftMax: 0.0, DConstMin: 0.60, DPairMax: 0.05},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := tt.in
			tt.cfg.ApplyThresholds(&th)
			if th != tt.want {
				t.Fatalf("ApplyThresholds() = %+v, want %+v", th, tt.want)
			}
		})
	}
}

// --- InstrumentOr: every set/unset field combination (🏗️ Production) ---

func TestInstrumentOr(t *testing.T) {
	def := internal.InstrumentConfig{
		Backend:       "",
		Model:         "",
		Temperature:   1.0,
		Samples:       10,
		Think:         false,
		NumCtx:        0,
		NumPredict:    0,
		SimThreshold:  0.95,
		Prompt:        "the prompt text",
		PromptVersion: "PromptV1",
	}

	tests := []struct {
		name string
		cfg  Config
		want internal.InstrumentConfig
	}{
		{
			name: "nil Instrument section returns def unchanged",
			cfg:  Config{},
			want: def,
		},
		{
			name: "empty Instrument section (all fields unset) returns def unchanged",
			cfg:  Config{Instrument: &Instrument{}},
			want: def,
		},
		{
			name: "only Backend set",
			cfg:  Config{Instrument: &Instrument{Backend: ptrS("ollama")}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.Backend = "ollama" }),
		},
		{
			name: "only Model set",
			cfg:  Config{Instrument: &Instrument{Model: ptrS("qwen3-coder:30b")}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.Model = "qwen3-coder:30b" }),
		},
		{
			name: "only Temperature set",
			cfg:  Config{Instrument: &Instrument{Temperature: ptrF(0.5)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.Temperature = 0.5 }),
		},
		{
			name: "only Samples set",
			cfg:  Config{Instrument: &Instrument{Samples: ptrI(20)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.Samples = 20 }),
		},
		{
			name: "only Think set true",
			cfg:  Config{Instrument: &Instrument{Think: ptrB(true)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.Think = true }),
		},
		{
			name: "only NumCtx set",
			cfg:  Config{Instrument: &Instrument{NumCtx: ptrI(8192)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.NumCtx = 8192 }),
		},
		{
			name: "only NumPredict set",
			cfg:  Config{Instrument: &Instrument{NumPredict: ptrI(2048)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.NumPredict = 2048 }),
		},
		{
			name: "only SimThreshold set",
			cfg:  Config{Instrument: &Instrument{SimThreshold: ptrF(0.8)}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) { c.SimThreshold = 0.8 }),
		},
		{
			name: "all fields set, including explicit zero/false",
			cfg: Config{Instrument: &Instrument{
				Backend:      ptrS("ollama"),
				Model:        ptrS("m"),
				Temperature:  ptrF(0.0),
				Samples:      ptrI(2),
				Think:        ptrB(false),
				NumCtx:       ptrI(4096),
				NumPredict:   ptrI(512),
				SimThreshold: ptrF(0.0),
			}},
			want: withInstrument(def, func(c *internal.InstrumentConfig) {
				c.Backend = "ollama"
				c.Model = "m"
				c.Temperature = 0.0
				c.Samples = 2
				c.Think = false
				c.NumCtx = 4096
				c.NumPredict = 512
				c.SimThreshold = 0.0
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.InstrumentOr(def)
			if got != tt.want {
				t.Fatalf("InstrumentOr() = %+v, want %+v", got, tt.want)
			}
			if got.Prompt != def.Prompt || got.PromptVersion != def.PromptVersion {
				t.Fatalf("InstrumentOr() must never touch Prompt/PromptVersion, got %+v", got)
			}
		})
	}
}

// withInstrument returns a copy of def with mutate applied — keeps the
// table above's "want" values expressed as a diff from def rather than a
// fully spelled-out struct literal per case.
func withInstrument(def internal.InstrumentConfig, mutate func(*internal.InstrumentConfig)) internal.InstrumentConfig {
	mutate(&def)
	return def
}
