// Package scenario executes a committed scenario against a released kno
// binary and compares what came back to what is committed.
//
// Two kinds of assertion, and they blame different things:
//
//   - A projection over the `--json` document. `expected/<stage>.json` is a
//     SUBSET of the real document — the fields a recipe's prose actually makes
//     a claim about. A full golden would churn on every run id, timestamp and
//     duration, so it would be regenerated reflexively and rubber-stamped,
//     which is a golden file that has stopped being a test. With a projection,
//     an additive CLI change passes (the recipe's claim is untouched) and a
//     removed or renamed field fails (the claim just became false).
//
//   - A quotation over the rendered text. Where a recipe quotes CLI output,
//     the assertion is that the exact phrase appears. If the CLI reformats,
//     the thing that must change is the prose — and that is the correct
//     direction of blame.
package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Scenario is one committed scenario directory.
type Scenario struct {
	Dir  string
	Name string
}

// Load opens a scenario directory and checks that the files the contract
// requires are present.
func Load(dir string) (*Scenario, error) {
	for _, must := range []string{
		"run.sh", "README.md", "DATA-PROVENANCE.md",
		filepath.Join("evals", "cases.jsonl"),
		filepath.Join("pool", "pool.jsonl"),
	} {
		if _, err := os.Stat(filepath.Join(dir, must)); err != nil {
			return nil, fmt.Errorf("scenario %s is missing %s: %w", dir, must, err)
		}
	}
	return &Scenario{Dir: dir, Name: filepath.Base(dir)}, nil
}

// Artifacts is what one execution of run.sh produced.
type Artifacts struct {
	Dir string
}

// Run executes the scenario's run.sh with `kno` resolved from knoPath, into a
// fresh directory under parent.
//
// run.sh is invoked exactly as a reader would invoke it — `sh run.sh <dir>` —
// so the bytes CI runs are the bytes the recipe shows.
func (s *Scenario) Run(knoPath, parent string) (*Artifacts, error) {
	out, err := os.MkdirTemp(parent, "run-")
	if err != nil {
		return nil, fmt.Errorf("make artifact dir: %w", err)
	}
	script := filepath.Join(s.Dir, "run.sh")
	cmd := exec.Command("sh", script, out) //nolint:gosec // script path is a repo path, not input
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	// The scenario must need nothing from the environment but a PATH that
	// finds kno. Handing it a curated environment is how "works on my machine
	// because I had KNO_DB set" is made impossible.
	cmd.Env = []string{
		"PATH=" + filepath.Dir(knoPath) + string(os.PathListSeparator) + "/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + out,
		"TMPDIR=" + out,
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run.sh failed: %w", err)
	}
	return &Artifacts{Dir: out}, nil
}

// Expectation is one stage's committed expectations.
type Expectation struct {
	Stage      string
	Projection map[string]any // nil when the stage has no --json form
	Quotations []string
}

// Expectations reads `expected/` for the scenario.
//
// `expected/quotations.json` maps a stage to the exact phrases a recipe quotes
// from that stage's rendered output.
func (s *Scenario) Expectations() ([]Expectation, error) {
	quotes := map[string][]string{}
	qPath := filepath.Join(s.Dir, "expected", "quotations.json")
	if b, err := os.ReadFile(qPath); err == nil { //nolint:gosec // repo path
		if err := json.Unmarshal(b, &quotes); err != nil {
			return nil, fmt.Errorf("%s: %w", qPath, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", qPath, err)
	}

	stages := map[string]bool{}
	for st := range quotes {
		stages[st] = true
	}
	files, err := filepath.Glob(filepath.Join(s.Dir, "expected", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob expected: %w", err)
	}
	projections := map[string]map[string]any{}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		if name == "quotations" {
			continue
		}
		b, err := os.ReadFile(f) //nolint:gosec // repo path
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		var doc map[string]any
		if err := decode(b, &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		projections[name] = doc
		stages[name] = true
	}

	names := make([]string, 0, len(stages))
	for st := range stages {
		names = append(names, st)
	}
	sort.Strings(names)
	out := make([]Expectation, 0, len(names))
	for _, st := range names {
		out = append(out, Expectation{Stage: st, Projection: projections[st], Quotations: quotes[st]})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("scenario %s has no expectations: a scenario that asserts nothing is not a scenario", s.Name)
	}
	return out, nil
}

// Check compares one execution's artifacts against the committed
// expectations, reporting every mismatch rather than the first.
func (s *Scenario) Check(a *Artifacts) ([]string, error) {
	exps, err := s.Expectations()
	if err != nil {
		return nil, err
	}
	var findings []string
	for _, e := range exps {
		if e.Projection != nil {
			b, err := os.ReadFile(filepath.Join(a.Dir, e.Stage+".json")) //nolint:gosec // temp path we created
			if err != nil {
				findings = append(findings, fmt.Sprintf("%s/%s: no --json output was captured: %v", s.Name, e.Stage, err))
				continue
			}
			var got map[string]any
			if err := decode(b, &got); err != nil {
				findings = append(findings, fmt.Sprintf("%s/%s: --json output did not parse: %v", s.Name, e.Stage, err))
				continue
			}
			for _, m := range Subset(e.Projection, got, e.Stage) {
				findings = append(findings, s.Name+"/"+m)
			}
		}
		if len(e.Quotations) > 0 {
			b, err := os.ReadFile(filepath.Join(a.Dir, e.Stage+".txt")) //nolint:gosec // temp path we created
			if err != nil {
				findings = append(findings, fmt.Sprintf("%s/%s: no rendered output was captured: %v", s.Name, e.Stage, err))
				continue
			}
			text := string(b)
			for _, q := range e.Quotations {
				if !strings.Contains(text, q) {
					findings = append(findings, fmt.Sprintf(
						"%s/%s: the recipe quotes %q but the binary did not print it — fix the prose, not the assertion",
						s.Name, e.Stage, q))
				}
			}
		}
	}
	return findings, nil
}

// Update rewrites `expected/<stage>.json` from an execution, preserving each
// file's existing key set.
//
// Preserving the shape is the point: `make update-expected` refreshes VALUES a
// human then reviews; it must not silently widen a projection into a full
// golden, because a projection that grows is a projection that starts
// churning, and a churning golden gets rubber-stamped.
func (s *Scenario) Update(a *Artifacts) error {
	exps, err := s.Expectations()
	if err != nil {
		return err
	}
	for _, e := range exps {
		if e.Projection == nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(a.Dir, e.Stage+".json")) //nolint:gosec // temp path we created
		if err != nil {
			return fmt.Errorf("%s: %w", e.Stage, err)
		}
		var got map[string]any
		if err := decode(b, &got); err != nil {
			return fmt.Errorf("%s: %w", e.Stage, err)
		}
		pruned, err := prune(e.Projection, got)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Stage, err)
		}
		enc, err := json.MarshalIndent(pruned, "", "  ")
		if err != nil {
			return fmt.Errorf("%s: %w", e.Stage, err)
		}
		path := filepath.Join(s.Dir, "expected", e.Stage+".json")
		if err := os.WriteFile(path, append(enc, '\n'), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

// CompareRuns byte-compares the captured artifacts of two executions.
//
// Anything varying between two runs of the same scenario on the same binary is
// a kno determinism bug, and the scenario surfaces it — a flakiness detector
// for free.
func CompareRuns(a, b *Artifacts) ([]string, error) {
	entries, err := os.ReadDir(a.Dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", a.Dir, err)
	}
	var findings []string
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".err") {
			continue
		}
		x, err := os.ReadFile(filepath.Join(a.Dir, e.Name())) //nolint:gosec // temp path we created
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		y, err := os.ReadFile(filepath.Join(b.Dir, e.Name())) //nolint:gosec // temp path we created
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		if !bytes.Equal(x, y) {
			findings = append(findings, fmt.Sprintf(
				"%s differs between two runs of the same binary — this is a kno determinism bug, not a docs finding", e.Name()))
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// decode parses JSON preserving numeric literals exactly, so a projection
// compares the bytes the binary printed rather than a float64 round-trip.
func decode(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
