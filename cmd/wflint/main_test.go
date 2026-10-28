package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const td = "../../internal/wflint/testdata/"

func runArgs(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRun_ViolationsExit1(t *testing.T) {
	code, out, _ := runArgs(td + "elements.templ.txt")
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stdout:\n%s", code, out)
	}
	for _, s := range []string{"element:", "→ use pageheader"} {
		if !strings.Contains(out, s) {
			t.Errorf("stdout missing %q:\n%s", s, out)
		}
	}
}

func TestRun_CleanExit0(t *testing.T) {
	code, out, errs := runArgs(td + "clean.templ.txt")
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stdout:\n%s\nstderr:\n%s", code, out, errs)
	}
	if !strings.Contains(out, "0 violations") {
		t.Errorf("stdout missing %q:\n%s", "0 violations", out)
	}
}

func TestRun_FenceSummary(t *testing.T) {
	_, out, _ := runArgs(td + "fenced.templ.txt")
	for _, s := range []string{"fence ", "landing page"} {
		if !strings.Contains(out, s) {
			t.Errorf("stdout missing %q:\n%s", s, out)
		}
	}
}

func TestRun_BadPathExit2(t *testing.T) {
	code, _, errs := runArgs(td + "does-not-exist.templ.txt")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.HasPrefix(errs, "wflint: ") || strings.HasPrefix(errs, "wflint: wflint:") {
		t.Errorf("stderr = %q, want one wflint: prefix", errs)
	}
}

// TestRun_DotDotDot: "./..." (as in go vet ./...) means the current
// directory, walked recursively.
func TestRun_DotDotDot(t *testing.T) {
	clean, err := os.ReadFile(td + "clean.templ.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "clean.templ"), clean, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, arg := range []string{"./...", "...", "sub/..."} {
		code, out, errs := runArgs(arg)
		if code != 0 {
			t.Errorf("%s: exit = %d, want 0; stdout:\n%s\nstderr:\n%s", arg, code, out, errs)
		}
		if !strings.Contains(out, "0 violations") {
			t.Errorf("%s: stdout = %q", arg, out)
		}
	}
}
