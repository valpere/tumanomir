package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/dispersion"
	"github.com/valpere/tumanomir/internal/instrument"
)

// keepFake drives 4 slots: slot0 = invalid then Foo; slot1 = 3 timeouts
// (discarded); slot2 = Bar; slot3 = Foo. So K=3 accepted, M=1 invalid,
// T=3 timed out, discarded=1.
func keepFake() *fakeGenerator {
	return &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		switch call {
		case 0:
			return genOK([]byte("no go here"))
		case 1:
			return genOK(goBlock(testSrcFoo))
		case 2, 3, 4:
			return instrument.Generation{}, fmt.Errorf("slow: %w", context.DeadlineExceeded)
		case 5:
			return genOK(goBlock(testSrcBar))
		default:
			return genOK(goBlock(testSrcFoo))
		}
	}}
}

func keepSpec(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(p, []byte("[REQ-X-01] x\n-> [FUN-X-01] y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

var keepBase = []string{"--instrument", "ollama:m", "--samples", "4", "--num-ctx", "4096", "--num-predict", "512", "--format", "json", "--d-pair-max", "0.9"}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// runKeep runs measure with extra args and returns stdout, stderr, code.
func runKeep(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	chdir(t, t.TempDir()) // no ambient .tumanomir.yaml
	full := append(append([]string{}, keepBase...), args...)
	return func() (string, string, int) {
		so, se, code := captureBoth(t, func() int {
			return runMeasureImpl(full, func(internal.InstrumentConfig) instrument.Generator { return keepFake() })
		})
		return so, se, code
	}()
}

func TestKeepSamplesOffWritesNothingAndStdoutUnchanged(t *testing.T) {
	spec := keepSpec(t)
	off, offErr, code := runKeep(t, spec)
	if code != 0 || strings.Contains(offErr, "samples") {
		t.Fatalf("code=%d stderr=%q", code, offErr)
	}
	root := t.TempDir()
	on, onErr, code := runKeep(t, "--keep-samples", root, spec)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, onErr)
	}
	if on != off {
		t.Fatalf("stdout changed by --keep-samples:\n%s\nvs\n%s", on, off)
	}
	if len(listDir(t, root)) != 1 {
		t.Fatalf("want exactly one run dir in %s", root)
	}
}

func TestKeepSamplesContentsAndReanalysis(t *testing.T) {
	spec := keepSpec(t)
	root := t.TempDir()
	out, errOut, code := runKeep(t, "--keep-samples", root, spec)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
	runs := listDir(t, root)
	if len(runs) != 1 {
		t.Fatalf("runs = %v", runs)
	}
	run := filepath.Join(root, runs[0])
	if !strings.Contains(errOut, run) {
		t.Fatalf("stderr %q must name created path %q", errOut, run)
	}
	want := []string{"00.go", "01.go", "02.go", "instrument.json", "s00-a0.discard.txt", "s01-a0.timeout.txt", "s01-a1.timeout.txt", "s01-a2.timeout.txt"}
	if got := listDir(t, run); !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %v, want %v", got, want)
	}

	var rep struct {
		Result struct {
			Dispersion internal.DispersionResult `json:"dispersion"`
			Config     internal.InstrumentConfig `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if rep.Result.Dispersion.Discarded != 1 {
		t.Fatalf("discarded = %d, want 1", rep.Result.Dispersion.Discarded)
	}
	b, _ := os.ReadFile(filepath.Join(run, "instrument.json"))
	var cfg internal.InstrumentConfig
	if err := json.Unmarshal(b, &cfg); err != nil || !reflect.DeepEqual(cfg, rep.Result.Config) {
		t.Fatalf("instrument.json %+v (err %v) != report config %+v", cfg, err, rep.Result.Config)
	}

	var srcs [][]byte
	for _, n := range listDir(t, run) {
		if strings.HasSuffix(n, ".go") {
			s, _ := os.ReadFile(filepath.Join(run, n))
			srcs = append(srcs, s)
		}
	}
	got := dispersion.Analyze(srcs, rep.Result.Config.SimThreshold)
	got.Discarded = rep.Result.Dispersion.Discarded
	if !reflect.DeepEqual(got, rep.Result.Dispersion) {
		t.Fatalf("re-analysis %+v != report %+v", got, rep.Result.Dispersion)
	}
}

func TestKeepSamplesWriteFailureKeepsExitCode(t *testing.T) {
	spec := keepSpec(t)
	_, _, base := runKeep(t, spec)
	blocker := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := runKeep(t, "--keep-samples", filepath.Join(blocker, "x"), spec)
	if code != base {
		t.Fatalf("code = %d, want %d", code, base)
	}
	if !strings.Contains(errOut, "warning") || !strings.Contains(errOut, "keep-samples") {
		t.Fatalf("want keep-samples warning, got %q", errOut)
	}
}

func TestKeepSamplesRerunNeverMixes(t *testing.T) {
	spec := keepSpec(t)
	root := t.TempDir()
	for i := 0; i < 2; i++ {
		if _, e, code := runKeep(t, "--keep-samples", root, spec); code != 0 {
			t.Fatalf("code=%d %s", code, e)
		}
	}
	runs := listDir(t, root)
	if len(runs) != 2 {
		t.Fatalf("runs = %v, want 2 distinct dirs", runs)
	}
	for _, r := range runs {
		if n := len(listDir(t, filepath.Join(root, r))); n != 8 {
			t.Fatalf("%s has %d files, want 8", r, n)
		}
	}
}

func TestKeepSamplesNothingWrittenOnHardFailure(t *testing.T) {
	spec := keepSpec(t)
	root := t.TempDir()
	chdir(t, t.TempDir())
	args := append(append([]string{}, keepBase...), "--keep-samples", root, spec)
	_, code := captureStderr(t, func() int {
		return runMeasureImpl(args, func(internal.InstrumentConfig) instrument.Generator {
			return &fakeGenerator{fn: func(int) (instrument.Generation, error) { return instrument.Generation{}, errFakeGenerate }}
		})
	})
	if code != 2 || len(listDir(t, root)) != 0 {
		t.Fatalf("code=%d entries=%v", code, listDir(t, root))
	}
}

func TestKeepSamplesConfigAndOverride(t *testing.T) {
	spec := keepSpec(t)
	cwd := t.TempDir()
	cfgRoot, cliRoot := t.TempDir(), t.TempDir()
	chdir(t, cwd)
	if err := os.WriteFile(defaultConfigPath, []byte("keep_samples: "+cfgRoot+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(extra ...string) {
		args := append(append(append([]string{}, keepBase...), extra...), spec)
		if _, code := captureStderr(t, func() int {
			return runMeasureImpl(args, func(internal.InstrumentConfig) instrument.Generator { return keepFake() })
		}); code != 0 {
			t.Fatalf("code = %d", code)
		}
	}
	run()
	if len(listDir(t, cfgRoot)) != 1 {
		t.Fatalf("config keep_samples not honored")
	}
	run("--keep-samples", cliRoot)
	if len(listDir(t, cliRoot)) != 1 || len(listDir(t, cfgRoot)) != 1 {
		t.Fatalf("CLI flag must override config: cli=%v cfg=%v", listDir(t, cliRoot), listDir(t, cfgRoot))
	}
	run("--keep-samples=")
	if len(listDir(t, cfgRoot)) != 1 {
		t.Fatalf("empty flag must disable configured dir")
	}
}

func TestGateKeepSamples(t *testing.T) {
	spec := keepSpec(t)
	chdir(t, t.TempDir())
	root := t.TempDir()
	args := []string{"--instrument", "ollama:m", "--samples", "4", "--num-ctx", "4096", "--num-predict", "512", "--d-pair-max", "0.9", "--keep-samples", root, spec}
	if _, code := captureBoth3(t, func() int {
		return runGateImpl(args, func(internal.InstrumentConfig) instrument.Generator { return keepFake() })
	}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	runs := listDir(t, root)
	if len(runs) != 1 || len(listDir(t, filepath.Join(root, runs[0]))) != 8 {
		t.Fatalf("gate wrote %v", runs)
	}
}

func captureBoth3(t *testing.T, fn func() int) (string, int) {
	t.Helper()
	_, se, code := captureBoth(t, fn)
	return se, code
}

func TestGateKeepSamplesWithoutInstrumentIsContradiction(t *testing.T) {
	spec := keepSpec(t)
	chdir(t, t.TempDir())
	errOut, code := captureStderr(t, func() int {
		return runGateImpl([]string{"--keep-samples", t.TempDir(), spec}, func(internal.InstrumentConfig) instrument.Generator {
			t.Fatal("newGen must not be called")
			return nil
		})
	})
	if code != 2 || !strings.Contains(errOut, "--keep-samples") || !strings.Contains(errOut, "REQ-GATE-02") {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}
