package scenario

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Subset asserts that every path present in `want` is present in `got` with
// the same value, and reports every mismatch it finds.
//
// The asymmetry is the whole design. A field in `got` that `want` does not
// mention is ignored, so an additive `--json` change passes — correct, because
// no recipe made a claim about it. A field in `want` that `got` lacks, or
// whose value moved, is a finding — correct, because a recipe's claim just
// became false.
func Subset(want, got map[string]any, path string) []string {
	return subsetValue(want, got, path)
}

//nolint:gocognit // a type switch over the six JSON kinds; splitting it would scatter the contract.
func subsetValue(want, got any, path string) []string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s: expected an object, got %s", path, kind(got))}
		}
		var out []string
		for _, k := range sortedKeys(w) {
			gv, present := g[k]
			if !present {
				out = append(out, fmt.Sprintf("%s.%s: field is gone from the --json document (renamed or removed)", path, k))
				continue
			}
			out = append(out, subsetValue(w[k], gv, path+"."+k)...)
		}
		return out

	case []any:
		g, ok := got.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: expected an array, got %s", path, kind(got))}
		}
		if len(w) != len(g) {
			return []string{fmt.Sprintf("%s: expected %d element(s), got %d", path, len(w), len(g))}
		}
		var out []string
		for i := range w {
			out = append(out, subsetValue(w[i], g[i], fmt.Sprintf("%s[%d]", path, i))...)
		}
		return out

	case json.Number:
		g, ok := got.(json.Number)
		if !ok {
			return []string{fmt.Sprintf("%s: expected the number %s, got %s", path, w.String(), kind(got))}
		}
		if !numbersMatch(w, g) {
			return []string{fmt.Sprintf("%s: expected %s, got %s", path, w.String(), g.String())}
		}
		return nil

	case nil:
		if got != nil {
			return []string{fmt.Sprintf("%s: expected null, got %v", path, got)}
		}
		return nil

	default:
		if want != got {
			return []string{fmt.Sprintf("%s: expected %v, got %v", path, want, got)}
		}
		return nil
	}
}

// numbersMatch compares a projected number against a real one AT THE PRECISION
// THE PROJECTION DECLARES.
//
// An integer is exact: `"scored": 8` means eight. A number written with a
// decimal point is compared after rounding both sides to that many fractional
// digits, so `"low": -0.3960` asserts the interval bound to four places and no
// further.
//
// This is not laziness, it is the assertion the recipe actually makes. Two
// findings drove it, on the very first CI run of this repository:
//
//	FAIL support-refunds/value.valuations[0].low:
//	  expected -0.39597252156206514, got -0.39597252156200174
//
// The scenario is bit-for-bit reproducible on one machine — two runs on
// darwin/arm64 produce identical bytes, and so do two runs on linux/amd64 —
// but the two platforms disagree from the 12th significant digit, which is a
// floating-point difference in an iterative computation, not a docs finding
// and not something a recipe ever claimed. Asserting all seventeen digits
// would make every expectation a statement about the runner's libm, so the
// scenario would be red on one architecture forever and the projection would
// be regenerated per-platform, which is a golden file that has stopped being a
// test.
//
// Four places is also exactly what the CLI renders (`+0.0000 [-0.3960,
// +0.3960]`) and exactly what the recipe's prose quotes. The projection and
// the quotation now claim the same thing.
//
// A widening in the OTHER direction still fails, which is the property that
// matters: if a bound moves at the fourth decimal, the recipe's claim about
// the interval became false and the page goes red.
func numbersMatch(want, got json.Number) bool {
	ws, gs := want.String(), got.String()
	if ws == gs {
		return true
	}
	dot := strings.IndexByte(ws, '.')
	if dot < 0 {
		// An integer projection is exact. `"scored": 8` is a count, and a
		// count that moved is always a finding.
		return false
	}
	digits := len(ws) - dot - 1
	wf, err := want.Float64()
	if err != nil {
		return false
	}
	gf, err := got.Float64()
	if err != nil {
		return false
	}
	return strconv.FormatFloat(wf, 'f', digits, 64) == strconv.FormatFloat(gf, 'f', digits, 64)
}

// roundTo renders got at the precision want declares, so `make update-expected`
// refreshes a value without widening its precision.
func roundTo(want, got json.Number) json.Number {
	ws := want.String()
	dot := strings.IndexByte(ws, '.')
	if dot < 0 {
		return got
	}
	gf, err := got.Float64()
	if err != nil {
		return got
	}
	return json.Number(strconv.FormatFloat(gf, 'f', len(ws)-dot-1, 64))
}

// prune projects `got` down to the shape of `want`. It is what
// `make update-expected` writes, so regeneration refreshes values without
// widening the projection.
func prune(want, got any) (any, error) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected an object, got %s", kind(got))
		}
		out := make(map[string]any, len(w))
		for k, wv := range w {
			gv, present := g[k]
			if !present {
				return nil, fmt.Errorf("field %q is gone from the --json document; regeneration cannot invent it — decide whether the recipe's claim survives", k)
			}
			p, err := prune(wv, gv)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = p
		}
		return out, nil
	case []any:
		g, ok := got.([]any)
		if !ok {
			return nil, fmt.Errorf("expected an array, got %s", kind(got))
		}
		out := make([]any, 0, len(g))
		for i := range g {
			shape := any(nil)
			if i < len(w) {
				shape = w[i]
			} else if len(w) > 0 {
				shape = w[0]
			}
			if shape == nil {
				out = append(out, g[i])
				continue
			}
			p, err := prune(shape, g[i])
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
		return out, nil
	case json.Number:
		g, ok := got.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected a number, got %s", kind(got))
		}
		return roundTo(w, g), nil
	default:
		return got, nil
	}
}

func kind(v any) string {
	switch v.(type) {
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
