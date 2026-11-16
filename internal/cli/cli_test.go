package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func TestDashE(t *testing.T) {
	code, out, errb := run(t, []string{"-e", "(+ 1 2)"}, "")
	if code != 0 || out != "3\n" || errb != "" {
		t.Errorf("code=%d out=%q err=%q", code, out, errb)
	}
}

func TestDashEError(t *testing.T) {
	code, _, errb := run(t, []string{"-e", "(undefined-var)"}, "")
	if code != 1 || !strings.Contains(errb, "unbound symbol") {
		t.Errorf("code=%d err=%q", code, errb)
	}
}

// TestGoldenPrograms runs every testdata/programs/*.lark through `lark run`
// and compares stdout with the sibling .out file.
func TestGoldenPrograms(t *testing.T) {
	files, err := filepath.Glob("../../testdata/programs/*.lark")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v (files=%v)", err, files)
	}
	for _, f := range files {
		want, err := os.ReadFile(strings.TrimSuffix(f, ".lark") + ".out")
		if err != nil {
			t.Fatalf("read %s.out: %v", f, err)
		}
		code, out, errb := run(t, []string{"run", f}, "")
		if code != 0 {
			t.Errorf("run %s: code=%d err=%q", f, code, errb)
			continue
		}
		if out != string(want) {
			t.Errorf("run %s: got %q, want %q", f, out, want)
		}
	}
}

func TestParseAndEmit(t *testing.T) {
	code, out, _ := run(t, []string{"parse", "../../testdata/sample.lark"}, "")
	if code != 0 {
		t.Fatalf("parse: code=%d", code)
	}
	want := "(define (square x) (* x x))\n(quote (1 2 3))\n#(a \"b\" 3.5)\n"
	if out != want {
		t.Errorf("parse out=%q want %q", out, want)
	}
	code, out, _ = run(t, []string{"emit", "../../testdata/sample.lark"}, "")
	if code != 0 || !strings.Contains(out, "(define (square x)") {
		t.Errorf("emit: code=%d out=%q", code, out)
	}
}

func TestREPL(t *testing.T) {
	code, out, errb := run(t, nil, "(+ 1\n 2)\n(define x 5)\nx\n")
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errb)
	}
	if out != "3\n()\n5\n" {
		t.Errorf("out=%q", out)
	}
}

func TestREPLError(t *testing.T) {
	code, _, errb := run(t, nil, "(car '())\n")
	if code != 0 || !strings.Contains(errb, "not a pair") {
		t.Errorf("code=%d err=%q", code, errb)
	}
}

func TestUsage(t *testing.T) {
	code, out, _ := run(t, []string{"--help"}, "")
	if code != 0 || !strings.Contains(out, "usage:") {
		t.Errorf("code=%d out=%q", code, out)
	}
	code, _, _ = run(t, []string{"bogus"}, "")
	if code != 2 {
		t.Errorf("bogus subcommand: code=%d", code)
	}
}
