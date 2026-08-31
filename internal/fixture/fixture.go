// Package fixture compares the three copies of the support-refunds fixture
// data that necessarily exist.
//
// The twelve Cases and three Pool assets live in three places:
//
//   - this repository, as scenarios/support-refunds/{evals/cases,pool/pool}.jsonl
//   - uknoAI/kno, embedded in the binary at cli/demodata/, so `kno demo` works
//     with no network — on a plane, in a locked-down CI runner, at a conference
//   - uknoAI/kno, typed line by line in tapes/quickstart.tape, because a VHS
//     tape records a terminal session and cannot read a file the viewer will
//     not see
//
// None of that duplication is a mistake, and none of it can be removed without
// losing the property that motivated it. What can be removed is the vigilance:
// the cost of duplication is paid by a detector rather than by whoever last
// remembered to check.
//
// This is the drift that shipped once already. The README of uknoAI/kno said
// refunds are "processed within 5 business days" while tapes/quickstart.tape
// said "issued", and nothing could tell you which was right.
package fixture

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Record is one fixture line: its id, and the exact bytes that expressed it.
//
// Raw is compared, not a re-marshalled form. Two records that decode to equal
// maps but differ in key order or spacing are still a divergence between a file
// a reader copies and a file CI runs, which is the whole subject.
type Record struct {
	ID  string
	Raw string
}

// Kind distinguishes the two fixture streams. A Case carries what the agent was
// asked and what it should answer; a Pool asset carries candidate knowledge.
type Kind int

const (
	Case Kind = iota
	Pool
)

func (k Kind) String() string {
	if k == Pool {
		return "pool"
	}
	return "cases"
}

// Records parses JSONL into records, preserving each line's exact bytes.
func Records(b []byte) ([]Record, error) {
	var out []Record
	for i, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		id, err := identify(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		out = append(out, Record{ID: id, Raw: line})
	}
	return out, nil
}

// tapeLine matches a VHS `Type` command carrying a JSON object. Anything else
// in the tape — Sleep, Enter, comments, the commands themselves — is not
// fixture data and is ignored.
var tapeLine = regexp.MustCompile(`(?m)^Type '(\{.*\})' Enter$`)

// FromTape extracts the fixture records a tape types, split by kind.
//
// Kind is decided by the record's own shape rather than by position in the
// file, so a tape that grows a case or reorders its two streams is compared
// correctly instead of being silently misaligned.
func FromTape(b []byte) (cases, pool []Record, err error) {
	for _, m := range tapeLine.FindAllStringSubmatch(string(b), -1) {
		raw := m[1]
		id, err := identify(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("tape: %w", err)
		}
		k, err := kindOf(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("tape: %s: %w", id, err)
		}
		if k == Pool {
			pool = append(pool, Record{ID: id, Raw: raw})
		} else {
			cases = append(cases, Record{ID: id, Raw: raw})
		}
	}
	return cases, pool, nil
}

func identify(line string) (string, error) {
	var probe struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return "", fmt.Errorf("not JSON: %w", err)
	}
	if probe.ID == "" {
		return "", fmt.Errorf("record has no id: %s", truncate(line))
	}
	return probe.ID, nil
}

func kindOf(line string) (Kind, error) {
	var probe struct {
		Expected *string `json:"expected"`
		Content  *string `json:"content"`
	}
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return Case, fmt.Errorf("not JSON: %w", err)
	}
	switch {
	case probe.Expected != nil && probe.Content == nil:
		return Case, nil
	case probe.Content != nil && probe.Expected == nil:
		return Pool, nil
	default:
		return Case, fmt.Errorf(
			"cannot tell a Case from a Pool asset: a Case has `expected`, a Pool asset has `content`, and this has %s",
			describe(probe.Expected != nil, probe.Content != nil))
	}
}

func describe(expected, content bool) string {
	if expected && content {
		return "both"
	}
	return "neither"
}

// Compare reports every way two copies of one stream disagree: records only one
// side has, and records both have whose bytes differ.
//
// `what` names the stream, `a` and `b` name where each copy came from, so a
// finding says which file to open.
func Compare(what, aName string, a []Record, bName string, b []Record) []string {
	index := func(rs []Record) map[string]string {
		m := make(map[string]string, len(rs))
		for _, r := range rs {
			m[r.ID] = r.Raw
		}
		return m
	}
	ai, bi := index(a), index(b)

	seen := map[string]bool{}
	var ids []string
	for _, rs := range [][]Record{a, b} {
		for _, r := range rs {
			if !seen[r.ID] {
				seen[r.ID] = true
				ids = append(ids, r.ID)
			}
		}
	}
	sort.Strings(ids)

	var findings []string
	for _, id := range ids {
		av, aok := ai[id]
		bv, bok := bi[id]
		switch {
		case aok && !bok:
			findings = append(findings, fmt.Sprintf(
				"FAIL fixtures %s: %s is in %s but not in %s", what, id, aName, bName))
		case bok && !aok:
			findings = append(findings, fmt.Sprintf(
				"FAIL fixtures %s: %s is in %s but not in %s", what, id, bName, aName))
		case av != bv:
			findings = append(findings, fmt.Sprintf(
				"FAIL fixtures %s: %s differs between %s and %s\n  %s: %s\n  %s: %s",
				what, id, aName, bName, aName, truncate(av), bName, truncate(bv)))
		}
	}
	return findings
}

func truncate(s string) string {
	const max = 160
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
