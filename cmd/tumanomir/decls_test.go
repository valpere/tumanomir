package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/valpere/tumanomir/internal"
	"github.com/valpere/tumanomir/internal/instrument"
)

// declsFake alternates two structurally different sources so Decls has
// non-unanimous slots to report.
func declsFake() *fakeGenerator {
	return &fakeGenerator{fn: func(call int) (instrument.Generation, error) {
		if call%2 == 0 {
			return genOK(goBlock(testSrcFoo))
		}
		return genOK(goBlock(testSrcBar))
	}}
}

func runDecls(t *testing.T, args ...string) (string, int) {
	t.Helper()
	chdir(t, t.TempDir())
	full := append(append([]string{}, keepBase...), args...)
	out, _, code := captureBoth(t, func() int {
		return runMeasureImpl(full, func(internal.InstrumentConfig) instrument.Generator { return declsFake() })
	})
	return out, code
}

// AC6: --format json --decls adds a `decls` array and changes nothing else.
func TestMeasureDeclsJSONOnlyAddsDecls(t *testing.T) {
	spec := keepSpec(t)
	off, _ := runDecls(t, spec)
	on, code := runDecls(t, "--decls", spec)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	var offM, onM map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(off), &offM); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(on), &onM); err != nil {
		t.Fatal(err)
	}
	if _, ok := offM["result"]["decls"]; ok {
		t.Fatal("decls must be absent without --decls")
	}
	var decls []internal.DeclAgreement
	if err := json.Unmarshal(onM["result"]["decls"], &decls); err != nil || len(decls) == 0 {
		t.Fatalf("decls = %s, err %v", onM["result"]["decls"], err)
	}
	delete(onM["result"], "decls")
	if !reflect.DeepEqual(offM, onM) {
		t.Fatalf("other fields changed:\n%s\n%s", off, on)
	}
}

// AC2/AC6: text output gains the block only with --decls.
func TestMeasureDeclsTextOnlyWithFlag(t *testing.T) {
	spec := keepSpec(t)
	off, _ := runDecls(t, "--format", "text", spec)
	on, _ := runDecls(t, "--format", "text", "--decls", spec)
	if strings.Contains(off, "declarations") {
		t.Fatalf("default output must be unchanged:\n%s", off)
	}
	if !strings.HasPrefix(on, off) || !strings.Contains(on, "declarations unanimous") {
		t.Fatalf("--decls must append a block to the default output:\n%s", on)
	}
}

// AC7: --decls never changes the exit code, even on a block verdict.
func TestMeasureDeclsKeepsBlockExitCode(t *testing.T) {
	spec := keepSpec(t)
	for _, f := range []string{"text", "json"} {
		_, offCode := runDecls(t, "--format", f, "--d-pair-max", "0.01", spec)
		_, onCode := runDecls(t, "--format", f, "--d-pair-max", "0.01", "--decls", spec)
		if offCode != 1 || onCode != 1 {
			t.Fatalf("%s: block exit codes off=%d on=%d, want 1/1", f, offCode, onCode)
		}
	}
}
