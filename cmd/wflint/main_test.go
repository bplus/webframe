package main

import (
	"bytes"
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
	if !strings.HasPrefix(errs, "wflint: ") {
		t.Errorf("stderr = %q, want wflint: prefix", errs)
	}
}
