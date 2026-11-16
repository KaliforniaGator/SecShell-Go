package cmdmap

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// expandArgs parses a command line, runs the expansion stage, and returns the
// resulting command name followed by its arguments — i.e. the fully expanded
// argv the executor would run.
func expandArgs(t *testing.T, input string) []string {
	t.Helper()
	chain, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString(%q): %v", input, err)
	}
	ExpandChain(chain)
	cmd := chain.Pipelines[0].Commands[0]
	out := []string{}
	if cmd.Name != "" {
		out = append(out, cmd.Name)
	}
	return append(out, cmd.Args...)
}

func TestParameterExpansion(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("EMPTY", "")
	os.Unsetenv("NOPE")
	lastExitStatus = 7

	cases := []struct {
		input string
		want  []string
	}{
		{`echo $FOO`, []string{"echo", "bar"}},
		{`echo ${FOO}`, []string{"echo", "bar"}},
		{`echo ${FOO}baz`, []string{"echo", "barbaz"}},
		{`echo "$FOO"`, []string{"echo", "bar"}},
		{`echo '$FOO'`, []string{"echo", "$FOO"}},        // single quotes suppress
		{`echo \$FOO`, []string{"echo", "$FOO"}},         // escaped suppresses
		{`echo ${NOPE:-fallback}`, []string{"echo", "fallback"}},
		{`echo ${EMPTY:-fallback}`, []string{"echo", "fallback"}}, // :- treats empty as unset
		{`echo ${EMPTY-fallback}`, []string{"echo"}},             // - keeps empty-but-set; unquoted empty drops
		{`echo ${#FOO}`, []string{"echo", "3"}},
		{`echo $?`, []string{"echo", "7"}},
		{`echo pre${FOO}post`, []string{"echo", "prebarpost"}},
	}
	for _, c := range cases {
		got := expandArgs(t, c.input)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("expand %q = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestWordSplitting(t *testing.T) {
	t.Setenv("WORDS", "a b c")
	// Unquoted expansion is word-split into separate arguments.
	if got, want := expandArgs(t, `echo $WORDS`), []string{"echo", "a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("unquoted split = %q, want %q", got, want)
	}
	// Quoted expansion stays a single argument.
	if got, want := expandArgs(t, `echo "$WORDS"`), []string{"echo", "a b c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("quoted no-split = %q, want %q", got, want)
	}
}

func TestBraceExpansion(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{`echo {a,b,c}`, []string{"echo", "a", "b", "c"}},
		{`echo pre{1,2}post`, []string{"echo", "pre1post", "pre2post"}},
		{`echo {1..4}`, []string{"echo", "1", "2", "3", "4"}},
		{`echo {3..1}`, []string{"echo", "3", "2", "1"}},
		{`echo x{a,b}{1,2}`, []string{"echo", "xa1", "xa2", "xb1", "xb2"}},
		{`echo "{a,b}"`, []string{"echo", "{a,b}"}}, // quoted: literal
		{`echo {single}`, []string{"echo", "{single}"}},
	}
	for _, c := range cases {
		got := expandArgs(t, c.input)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("brace %q = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestArithmeticExpansion(t *testing.T) {
	t.Setenv("N", "5")
	cases := []struct {
		input string
		want  string
	}{
		{`echo $((2+3))`, "5"},
		{`echo $((2+3*4))`, "14"},
		{`echo $(( (1+2)*3 ))`, "9"},
		{`echo $((10%3))`, "1"},
		{`echo $((N*2))`, "10"},
		{`echo $((-4 + 1))`, "-3"},
	}
	for _, c := range cases {
		got := expandArgs(t, c.input)
		if len(got) != 2 || got[1] != c.want {
			t.Errorf("arith %q = %q, want [echo %s]", c.input, got, c.want)
		}
	}
}

func TestCommandSubstitution(t *testing.T) {
	if _, err := exec.LookPath("printf"); err != nil {
		t.Skip("printf not available")
	}
	cases := []struct {
		input string
		want  []string
	}{
		{"echo $(printf hello)", []string{"echo", "hello"}},
		{"echo `printf hello`", []string{"echo", "hello"}},
		{"echo pre$(printf mid)post", []string{"echo", "premidpost"}},
		// Unquoted substitution is word-split; quoted is not.
		{`echo "$(printf 'a b')"`, []string{"echo", "a b"}},
		{`echo $(printf 'a b')`, []string{"echo", "a", "b"}},
	}
	for _, c := range cases {
		got := expandArgs(t, c.input)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("cmdsub %q = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestTildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got := expandArgs(t, "echo ~"); len(got) != 2 || got[1] != home {
		t.Errorf("echo ~ = %q, want [echo %s]", got, home)
	}
	if got := expandArgs(t, "echo ~/docs"); len(got) != 2 || got[1] != home+"/docs" {
		t.Errorf("echo ~/docs = %q, want [echo %s/docs]", got, home)
	}
	// Quoted tilde is literal.
	if got := expandArgs(t, `echo "~"`); len(got) != 2 || got[1] != "~" {
		t.Errorf(`echo "~" = %q, want [echo ~]`, got)
	}
}

func TestPathnameGlob(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.log"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	got := expandArgs(t, "echo *.txt")
	want := []string{"echo", "a.txt", "b.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("glob *.txt = %q, want %q", got, want)
	}
	// No match stays literal (nullglob off).
	if got := expandArgs(t, "echo *.md"); !reflect.DeepEqual(got, []string{"echo", "*.md"}) {
		t.Errorf("glob no-match = %q, want [echo *.md]", got)
	}
	// Quoted glob is literal.
	if got := expandArgs(t, `echo "*.txt"`); !reflect.DeepEqual(got, []string{"echo", "*.txt"}) {
		t.Errorf("quoted glob = %q, want [echo *.txt]", got)
	}
}

// TestBashParity cross-checks a representative set of expansions against real
// bash, which is the objective measure of "as good as bash". Skipped when bash
// is unavailable.
func TestBashParity(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	t.Setenv("FOO", "bar")
	inputs := []string{
		`echo $FOO`,
		`echo "$FOO"`,
		`echo '$FOO'`,
		`echo ${FOO}x`,
		`echo ${NOPE:-def}`,
		`echo $((2+3*4))`,
		`echo {a,b,c}`,
		`echo pre{1..3}`,
		`echo a"b"c`,
	}
	for _, in := range inputs {
		// Our expansion.
		ours := strings.Join(expandArgs(t, in)[1:], " ")
		// Bash's expansion of the same echo.
		cmd := exec.Command(bash, "-c", in)
		cmd.Env = append(os.Environ(), "FOO=bar")
		out, err := cmd.Output()
		if err != nil {
			t.Errorf("bash -c %q: %v", in, err)
			continue
		}
		want := strings.TrimRight(string(out), "\n")
		if ours != want {
			t.Errorf("parity %q: ours=%q bash=%q", in, ours, want)
		}
	}
}
