package appshell

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHTMXErrorsScript runs tools/htmx-errors-check.mjs, which loads
// static/js/htmx-errors.js in a bare Node context and checks which error
// answers it lets htmx swap (an XML body, Go's "<nil>", an unlabelled HTML
// fragment, a full document, ...). It needs node on PATH and no npm
// packages; without node it is skipped.
func TestHTMXErrorsScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "tools", "htmx-errors-check.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(node, script).CombinedOutput()
	if err != nil {
		t.Fatalf("htmx-errors-check failed: %v\n%s", err, out)
	}
}
