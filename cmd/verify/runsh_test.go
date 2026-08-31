package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// run.sh captures each stage's stderr to a file so that interleaving two
// streams cannot make a committed expectation depend on buffering. That is the
// right call, and it used to mean the two most likely first-run failures — no
// `kno`, or a `kno` too old — reached the reader as a stage heading and a bare
// exit code, with the diagnosis sitting in a file whose path is printed only on
// success.
//
// These tests need no released binary: a shim on PATH stands in for kno, so the
// guarantee holds even where `make test` would skip for want of KNO_BIN.

// shimDir writes an executable `kno` shim and returns its ABSOLUTE directory —
// absolute because `pass` chdirs before running a stage, which would strip the
// meaning out of a relative PATH entry.
func shimDir(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kno")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil { //nolint:gosec // it must be executable
		t.Fatalf("write shim: %v", err)
	}
	return dir
}

// runScenario runs the committed run.sh with a curated PATH and returns the
// exit code and everything the reader would have seen on the terminal.
func runScenario(t *testing.T, pathEnv string) (int, string) {
	t.Helper()
	script := filepath.Join(repoRoot(t), "scenarios", "support-refunds", "run.sh")
	cmd := exec.Command("sh", script, t.TempDir()) //nolint:gosec // a repo path, not input
	cmd.Env = []string{"PATH=" + pathEnv, "HOME=" + t.TempDir()}
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if err != nil {
		if !asExitError(err, &ee) {
			t.Fatalf("run run.sh: %v", err)
		}
		code = ee.ExitCode()
	}
	return code, string(out)
}

func asExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError) //nolint:errorlint // the shape is exactly this
	if ok {
		*target = ee
	}
	return ok
}

// TestRunShRefusesToStartWithoutKno asserts the missing-binary case says what
// is missing and how to get it, rather than exiting 127 into silence.
func TestRunShRefusesToStartWithoutKno(t *testing.T) {
	code, out := runScenario(t, "/usr/bin:/bin")

	if code != 127 {
		t.Errorf("exit code = %d, want 127\n%s", code, out)
	}
	for _, want := range []string{"no `kno` on PATH", "make install-kno"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not mention %q — the reader is told nothing:\n%s", want, out)
		}
	}
	if strings.Contains(out, "baseline") {
		t.Errorf("a stage ran despite there being no binary:\n%s", out)
	}
}

// TestRunShReplaysAFailingStagesStderr is the regression test for the finding
// itself: a stage that fails must put its captured stderr on the terminal. The
// shim answers `--version` and the first two stages, then fails the way a kno
// too old for `select` actually failed.
func TestRunShReplaysAFailingStagesStderr(t *testing.T) {
	dir := shimDir(t, `case "$1" in
  --version) echo "kno version 0.0.4 (shim)" ;;
  baseline|value) echo '{}' ;;
  *) echo >&2; echo 'error: unknown command "'"$1"'" for "kno"' >&2; exit 1 ;;
esac
`)
	code, out := runScenario(t, dir+":/usr/bin:/bin")

	if code != 1 {
		t.Errorf("exit code = %d, want 1 (the stage's own code)\n%s", code, out)
	}
	if !strings.Contains(out, `error: unknown command "select" for "kno"`) {
		t.Errorf("the failing stage's stderr never reached the terminal:\n%s", out)
	}
	if !strings.Contains(out, "select FAILED") {
		t.Errorf("output does not name the stage that failed:\n%s", out)
	}
	// Reporting the binary is what makes a version mismatch self-diagnosing.
	if !strings.Contains(out, "kno version 0.0.4 (shim)") {
		t.Errorf("run.sh did not report which kno it used:\n%s", out)
	}
}
