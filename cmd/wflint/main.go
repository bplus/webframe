// Command wflint runs the webframe UI lint over templ, Go, and CSS sources and
// reports code that bypasses the component library. Consumers run it from
// `make check`.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/bplus/webframe/internal/wflint"
)

// run lints the paths in args (default ".") and writes the report to stdout.
// It returns the process exit code: 1 if any violation was found, 2 if the
// flags were bad or Lint failed (message on stderr), 0 otherwise.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wflint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	rep, err := wflint.Lint(paths)
	if err != nil {
		fmt.Fprintf(stderr, "wflint: %v\n", err)
		return 2
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintf(stderr, "wflint: %v\n", err)
			return 2
		}
	} else {
		for _, v := range rep.Violations {
			fmt.Fprintf(stdout, "%s:%d:%d: %s: %s\n", v.File, v.Line, v.Col, v.Rule, v.Message)
			if v.Hint != "" {
				fmt.Fprintf(stdout, "  → use %s\n", v.Hint)
			}
		}
		fmt.Fprintf(stdout, "wflint: %d violations, %d custom fences\n", len(rep.Violations), len(rep.Fences))
		for _, f := range rep.Fences {
			fmt.Fprintf(stdout, "  fence %s:%d (%s): %s\n", f.File, f.Line, f.Kind, f.Reason)
		}
	}

	if len(rep.Violations) > 0 {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
