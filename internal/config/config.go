// Package config loads the optional .tumanomir.yaml configuration file
// shared by the check/measure (and future gate) subcommands, so users
// don't have to repeat --k-drift-max/--instrument/... on every invocation
// (REQ-CFG-02). It depends only on internal (+ gopkg.in/yaml.v3) — never
// internal/metrics, internal/spec, internal/instrument, or
// internal/dispersion — keeping it as network-free as the packages it
// feeds into (see internal/nonetwork_test.go).
package config

import (
	"fmt"
	"os"

	"github.com/valpere/tumanomir/internal"
	"gopkg.in/yaml.v3"
)

// Config mirrors internal.Thresholds/internal.InstrumentConfig, minus
// Prompt/PromptVersion (deliberately non-configurable — reproducibility
// invariant, REQ-MSR-04). Every field is a pointer so yaml.v3's
// nil-on-absent semantics distinguish "unset in the file" from "explicit
// zero value" — required for CLI-flag > config > default precedence.
type Config struct {
	Thresholds *Thresholds `yaml:"thresholds"`
	Instrument *Instrument `yaml:"instrument"`
}

// Thresholds is the config-file counterpart of internal.Thresholds.
type Thresholds struct {
	KDriftMax *float64 `yaml:"k_drift_max"`
	DConstMin *float64 `yaml:"d_const_min"`
	DPairMax  *float64 `yaml:"d_pair_max"`
}

// Instrument is the config-file counterpart of internal.InstrumentConfig,
// minus Prompt/PromptVersion.
type Instrument struct {
	Backend      *string  `yaml:"backend"`
	Model        *string  `yaml:"model"`
	Temperature  *float64 `yaml:"temperature"`
	Samples      *int     `yaml:"samples"`
	Think        *bool    `yaml:"think"`
	NumCtx       *int     `yaml:"num_ctx"`
	NumPredict   *int     `yaml:"num_predict"`
	SimThreshold *float64 `yaml:"sim_threshold"`
}

// Load reads and parses the YAML config file at path. It returns an error
// if the file cannot be read (including "does not exist") or fails to
// parse — callers decide separately whether a missing file is fatal
// (explicit --config path) or silently ignorable (the default
// ./.tumanomir.yaml discovery), since that distinction depends on how the
// caller found path, not on anything Load itself can know.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// ApplyThresholds overwrites th's fields with any the config file sets
// explicitly, leaving fields it omits untouched — so callers seed th from
// internal.DefaultThresholds() first, call ApplyThresholds, and only then
// use th's fields as CLI flag defaults (giving CLI flag > config > default
// precedence for free from flag.Parse's own override behavior).
func (c Config) ApplyThresholds(th *internal.Thresholds) {
	if c.Thresholds == nil {
		return
	}
	if c.Thresholds.KDriftMax != nil {
		th.KDriftMax = *c.Thresholds.KDriftMax
	}
	if c.Thresholds.DConstMin != nil {
		th.DConstMin = *c.Thresholds.DConstMin
	}
	if c.Thresholds.DPairMax != nil {
		th.DPairMax = *c.Thresholds.DPairMax
	}
}

// InstrumentOr returns def with each field overridden by the config
// file's corresponding value when the config sets it. Prompt/PromptVersion
// are never touched (deliberately non-configurable, REQ-MSR-04) — def's
// values for those two fields pass through unchanged.
func (c Config) InstrumentOr(def internal.InstrumentConfig) internal.InstrumentConfig {
	if c.Instrument == nil {
		return def
	}
	i := c.Instrument
	if i.Backend != nil {
		def.Backend = *i.Backend
	}
	if i.Model != nil {
		def.Model = *i.Model
	}
	if i.Temperature != nil {
		def.Temperature = *i.Temperature
	}
	if i.Samples != nil {
		def.Samples = *i.Samples
	}
	if i.Think != nil {
		def.Think = *i.Think
	}
	if i.NumCtx != nil {
		def.NumCtx = *i.NumCtx
	}
	if i.NumPredict != nil {
		def.NumPredict = *i.NumPredict
	}
	if i.SimThreshold != nil {
		def.SimThreshold = *i.SimThreshold
	}
	return def
}
