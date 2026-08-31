package scenario

import (
	"encoding/json"
	"fmt"
	"sort"
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
		if !ok || g.String() != w.String() {
			return []string{fmt.Sprintf("%s: expected %s, got %v", path, w.String(), got)}
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
