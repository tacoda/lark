// Package cli implements the lark command: a REPL and run/-e/parse/emit
// subcommands.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tacoda/lark/internal/eval"
	"github.com/tacoda/lark/internal/sexp"
)

const usage = `usage: lark [repl]          start the REPL (default)
       lark run <file>     evaluate a file, print the last value
       lark -e <expr>      evaluate an expression, print the result
       lark parse <file>   print the AST of each top-level form
       lark emit <file>    parse and pretty-print each top-level form`

// Main runs the CLI. It returns the process exit code.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "repl" {
		return repl(stdin, stdout, stderr)
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, usage)
		return 0
	case "-e":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "lark -e: want exactly one expression")
			return 2
		}
		return runSrc(args[1], stdout, stderr)
	case "run":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "lark run: want exactly one file")
			return 2
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "lark: %v\n", err)
			return 1
		}
		return runSrc(string(data), stdout, stderr)
	case "parse":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "lark parse: want exactly one file")
			return 2
		}
		return dump(args[1], stdout, stderr, false)
	case "emit":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "lark emit: want exactly one file")
			return 2
		}
		return dump(args[1], stdout, stderr, true)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

func runSrc(src string, stdout, stderr io.Writer) int {
	v, err := eval.Run(src, eval.GlobalEnv())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, sexp.Pretty(v))
	return 0
}

func dump(path string, stdout, stderr io.Writer, pretty bool) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "lark: %v\n", err)
		return 1
	}
	forms, err := sexp.ParseAll(string(data))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	for _, f := range forms {
		if pretty {
			fmt.Fprintln(stdout, sexp.Pretty(f))
		} else {
			fmt.Fprintln(stdout, sexp.Write(f))
		}
	}
	return 0
}

func repl(stdin io.Reader, stdout, stderr io.Writer) int {
	term := false
	if f, ok := stdin.(*os.File); ok {
		if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
			term = true
		}
	}
	env := eval.GlobalEnv()
	prompt := func(cont bool) {
		if term {
			if cont {
				fmt.Fprint(stdout, "  ... ")
			} else {
				fmt.Fprint(stdout, "lark> ")
			}
		}
	}
	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	var buf strings.Builder
	prompt(false)
	for sc.Scan() {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(sc.Text())
		src := buf.String()
		form, err := sexp.Parse(src)
		if err != nil {
			if incomplete(err) {
				prompt(true)
				continue
			}
			fmt.Fprintf(stderr, "error: %v\n", err)
			buf.Reset()
			prompt(false)
			continue
		}
		res, err := eval.Eval(form, env)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
		} else {
			fmt.Fprintln(stdout, sexp.Pretty(res))
		}
		buf.Reset()
		prompt(false)
	}
	return 0
}

// incomplete reports whether a read error just means "need more input".
func incomplete(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unexpected end of input") ||
		strings.Contains(msg, "unterminated")
}
