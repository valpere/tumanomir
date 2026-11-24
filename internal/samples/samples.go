// Package samples persists a measure run's generations for offline audit
// and re-analysis (REQ-MSR-11). It imports only internal, report and the
// stdlib, so it stays network-free like the other non-instrument packages.
package samples

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/valpere/tumanomir/internal/report"
)

// Write creates a fresh run directory under root and writes the run's
// accepted Go blocks (NN.go, in Analyze order), discarded and timed-out
// attempts, and instrument.json. It returns the run directory. On any
// failure the partial run dir is removed, so a kept dir is always complete.
//
// WHY: os.MkdirTemp gives every run its own directory, so a rerun with the
// same spec and instrument can never overwrite or mix with earlier files.
// PURPOSE: the .go files, sorted by name, are exactly dispersion.Analyze's
// input, so D_pair can be reproduced offline.
func Write(root string, specContent []byte, mr report.MeasureResult) (string, error) {
	cfgJSON, err := json.MarshalIndent(mr.Config, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	specSum := sha256.Sum256(specContent)
	cfgSum := sha256.Sum256(cfgJSON)
	prefix := hex.EncodeToString(specSum[:])[:12] + "-" + hex.EncodeToString(cfgSum[:])[:8]
	dir, err := os.MkdirTemp(root, prefix+"-*")
	if err != nil {
		return "", err
	}

	if err := writeRun(dir, mr, cfgJSON); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func writeRun(dir string, mr report.MeasureResult, cfgJSON []byte) error {
	width := max(2, len(strconv.Itoa(mr.Config.Samples-1)))
	write := func(name string, data []byte) error {
		return os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}
	accepted := 0
	for _, r := range mr.Attempts {
		var name string
		switch r.Kind {
		case report.SampleAccepted:
			name = fmt.Sprintf("%0*d.go", width, accepted)
			accepted++
		case report.SampleDiscard:
			name = fmt.Sprintf("s%0*d-a%d.discard.txt", width, r.Slot, r.Attempt)
		case report.SampleTimeout:
			name = fmt.Sprintf("s%0*d-a%d.timeout.txt", width, r.Slot, r.Attempt)
		default:
			return fmt.Errorf("unknown sample kind %q", r.Kind)
		}
		if err := write(name, r.Data); err != nil {
			return err
		}
	}
	return write("instrument.json", append(cfgJSON, '\n'))
}
