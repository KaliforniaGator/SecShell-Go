package cmdmap

import (
	"io"
	"os"
	"strings"
	"testing"

	"secshell/jobs"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what was
// written. Used to verify real command output flows through the executor.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

func TestExecForLoopStdout(t *testing.T) {
	cleanupVars(t, "i")
	out := captureStdout(t, func() {
		runProg(t, "for i in 1 2 3; do echo line$i; done")
	})
	for _, want := range []string{"line1", "line2", "line3"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q missing %q", out, want)
		}
	}
}

// runProg parses and executes a program through the AST executor. Tests observe
// results via shell-variable side effects (assignments) rather than stdout, so
// no output capture is needed. External true/false live in /bin (a trusted dir)
// and so are permitted.
func runProg(t *testing.T, input string) {
	t.Helper()
	prog, err := ParseProgram(input)
	if err != nil {
		t.Fatalf("ParseProgram(%q): %v", input, err)
	}
	ExecuteProgram(prog, map[int]*jobs.Job{})
}

func wantVar(t *testing.T, name, want string) {
	t.Helper()
	if got, _ := ShellState.GetVar(name); got != want {
		t.Errorf("var %s = %q, want %q", name, got, want)
	}
}

func cleanupVars(t *testing.T, names ...string) {
	t.Cleanup(func() {
		for _, n := range names {
			ShellState.Unset(n)
		}
	})
}

func TestExecForLoop(t *testing.T) {
	cleanupVars(t, "LAST", "i", "SEEN")
	runProg(t, "for i in a b c; do LAST=$i; done")
	wantVar(t, "LAST", "c")

	runProg(t, "SEEN=; for i in 1 2 3; do SEEN=$SEEN$i; done")
	wantVar(t, "SEEN", "123")
}

func TestExecIf(t *testing.T) {
	cleanupVars(t, "R")
	runProg(t, "if true; then R=yes; fi")
	wantVar(t, "R", "yes")

	runProg(t, "if false; then R=yes; else R=no; fi")
	wantVar(t, "R", "no")

	runProg(t, "if false; then R=a; elif true; then R=b; else R=c; fi")
	wantVar(t, "R", "b")
}

func TestExecBreakContinue(t *testing.T) {
	cleanupVars(t, "GOT", "i")

	// break stops the loop after the first iteration.
	runProg(t, "for i in a b c; do GOT=$i; break; done")
	wantVar(t, "GOT", "a")

	// continue skips the rest of the body; GOT must never become "reached".
	runProg(t, "GOT=start; for i in a b c; do continue; GOT=reached; done")
	wantVar(t, "GOT", "start")
}

func TestExecWhileUntil(t *testing.T) {
	cleanupVars(t, "KEEP", "X")

	// false condition: body never runs.
	runProg(t, "KEEP=yes; while false; do KEEP=no; done")
	wantVar(t, "KEEP", "yes")

	// until true: body never runs.
	runProg(t, "X=init; until true; do X=ran; done")
	wantVar(t, "X", "init")
}

func TestExecCase(t *testing.T) {
	cleanupVars(t, "R")

	runProg(t, "case foo in bar) R=no;; foo) R=yes;; esac")
	wantVar(t, "R", "yes")

	runProg(t, "case zzz in a) R=a;; *) R=def;; esac")
	wantVar(t, "R", "def")

	// Glob pattern matching.
	runProg(t, "case hello in h*) R=glob;; esac")
	wantVar(t, "R", "glob")
}

func TestExecFunction(t *testing.T) {
	cleanupVars(t, "R")
	t.Cleanup(func() { ShellState.SetPositional(nil) })

	runProg(t, "greet() { R=$1; }; greet hello")
	wantVar(t, "R", "hello")

	// function keyword form, and positional params restore after the call.
	ShellState.SetPositional([]string{"outer"})
	runProg(t, "function f { R=$1; }; f inner")
	wantVar(t, "R", "inner")
	if got := ShellState.Positional(); len(got) != 1 || got[0] != "outer" {
		t.Errorf("positional not restored: %v", got)
	}
}

func TestExecSubshellIsolation(t *testing.T) {
	cleanupVars(t, "V")
	// A variable set inside a subshell does not leak out.
	runProg(t, "V=outer; (V=inner)")
	wantVar(t, "V", "outer")
}

func TestExecAndOrSequencing(t *testing.T) {
	cleanupVars(t, "R")
	// && runs the next only on success; || only on failure.
	runProg(t, "true && R=ran")
	wantVar(t, "R", "ran")

	runProg(t, "R=before; false && R=after")
	wantVar(t, "R", "before")

	runProg(t, "false || R=recovered")
	wantVar(t, "R", "recovered")
}
