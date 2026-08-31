package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The fixture data exists three times on purpose, and the detector is what
// makes that safe. `testdata/fixtures/` holds one directory per way the three
// copies can disagree, shaped like a checkout of uknoAI/kno.
//
// A detector that passes everything is the failure mode.

func fixturesFor(t *testing.T, variant string) []string {
	t.Helper()
	root := repoRoot(t)
	findings, err := cmdFixtures([]string{
		"--kno-src", filepath.Join(root, "cmd", "verify", "testdata", "fixtures", variant),
		"--scenario", filepath.Join(root, "scenarios", "support-refunds"),
	})
	if err != nil {
		t.Fatalf("fixtures %s: %v", variant, err)
	}
	return findings
}

func TestFixturesAcceptsThreeCopiesThatAgree(t *testing.T) {
	if f := fixturesFor(t, "good"); len(f) != 0 {
		t.Errorf("copies that agree produced findings:\n%s", strings.Join(f, "\n"))
	}
}

// TestFixturesCatchesTheHistoricDrift reproduces the bug this repository was
// built after: uknoAI/kno's prose said refunds are "processed" within five
// business days while tapes/quickstart.tape said "issued", and nothing could
// tell you which was right.
func TestFixturesCatchesTheHistoricDrift(t *testing.T) {
	findings := fixturesFor(t, "drift-processed")
	if len(findings) == 0 {
		t.Fatal("the processed/issued drift went undetected — this is the bug the detector exists for")
	}
	joined := strings.Join(findings, "\n")
	for _, want := range []string{"refund-01", "quickstart.tape", "processed"} {
		if !strings.Contains(joined, want) {
			t.Errorf("finding does not mention %q, so it does not say what to fix:\n%s", want, joined)
		}
	}
}

func TestFixturesCatchesARecordOnlyOneCopyHas(t *testing.T) {
	findings := fixturesFor(t, "missing-case")
	if len(findings) == 0 {
		t.Fatal("a case missing from the embedded copy went undetected")
	}
	joined := strings.Join(findings, "\n")
	if !strings.Contains(joined, "acct-02") || !strings.Contains(joined, "cli/demodata") {
		t.Errorf("finding does not name the record and the copy missing it:\n%s", joined)
	}
}

// TestFixturesCatchesReordering covers what a per-record comparison cannot see.
// Every record agrees; the files are still not interchangeable.
func TestFixturesCatchesReordering(t *testing.T) {
	findings := fixturesFor(t, "reordered")
	if len(findings) == 0 {
		t.Fatal("reordered copies went undetected: they are no longer byte-interchangeable")
	}
	if !strings.Contains(strings.Join(findings, "\n"), "byte-identical") {
		t.Errorf("finding does not explain that ordering is the problem:\n%s", strings.Join(findings, "\n"))
	}
}

func TestFixturesRefusesToRunWithoutASource(t *testing.T) {
	if _, err := cmdFixtures(nil); err == nil {
		t.Fatal("fixtures with no --kno-src should be an error, not a silent pass")
	}
}
