package cmdmap

import (
	"os"
	"reflect"
	"testing"

	"secshell/jobs"
)

func TestShellVarVsEnv(t *testing.T) {
	// A plain shell variable is visible to expansion but not exported to the env.
	ShellState.SetVar("SV", "shellval")
	t.Cleanup(func() { ShellState.Unset("SV") })

	if v, ok := ShellState.GetVar("SV"); !ok || v != "shellval" {
		t.Fatalf("GetVar(SV) = %q,%v want shellval,true", v, ok)
	}
	if _, inEnv := os.LookupEnv("SV"); inEnv {
		t.Errorf("SV should not be in the environment until exported")
	}

	// Exporting promotes it to the environment.
	ShellState.Export("SV", "", false)
	if v, inEnv := os.LookupEnv("SV"); !inEnv || v != "shellval" {
		t.Errorf("after export, env SV = %q,%v want shellval,true", v, inEnv)
	}

	// Unset clears both.
	ShellState.Unset("SV")
	if _, ok := ShellState.GetVar("SV"); ok {
		t.Errorf("SV still resolvable after Unset")
	}
}

func TestShellVarExpansion(t *testing.T) {
	ShellState.SetVar("GREETING", "hi")
	t.Cleanup(func() { ShellState.Unset("GREETING") })

	if got, want := expandArgs(t, "echo $GREETING"), []string{"echo", "hi"}; !reflect.DeepEqual(got, want) {
		t.Errorf("shell var expansion = %q, want %q", got, want)
	}
}

func TestPositionalParams(t *testing.T) {
	ShellState.SetPositional([]string{"one", "two", "three"})
	t.Cleanup(func() { ShellState.SetPositional(nil) })

	cases := []struct {
		input string
		want  []string
	}{
		{"echo $1", []string{"echo", "one"}},
		{"echo $3", []string{"echo", "three"}},
		{"echo $#", []string{"echo", "3"}},
		{`echo "$1 $2"`, []string{"echo", "one two"}},
	}
	for _, c := range cases {
		if got := expandArgs(t, c.input); !reflect.DeepEqual(got, c.want) {
			t.Errorf("expand %q = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestAssignmentParsing(t *testing.T) {
	if !isAssignmentWord("FOO=bar") || !isAssignmentWord("_x=") || isAssignmentWord("1BAD=x") || isAssignmentWord("noeq") {
		t.Errorf("isAssignmentWord classification wrong")
	}

	// Pure assignment.
	assigns, _, pure := splitAssignments(CommandNode{Name: "X=1", Args: []string{"Y=2"}})
	if !pure || len(assigns) != 2 || assigns[0] != (assignment{"X", "1"}) || assigns[1] != (assignment{"Y", "2"}) {
		t.Errorf("pure assignments = %+v pure=%v", assigns, pure)
	}

	// Prefix assignment before a command.
	assigns, rest, pure := splitAssignments(CommandNode{Name: "X=1", Args: []string{"echo", "hi"}})
	if pure || len(assigns) != 1 || rest.Name != "echo" || !reflect.DeepEqual(rest.Args, []string{"hi"}) {
		t.Errorf("prefix split: assigns=%+v rest=%+v pure=%v", assigns, rest, pure)
	}
}

func TestPureAssignmentSetsVar(t *testing.T) {
	ShellState.Unset("ASSIGNED")
	t.Cleanup(func() { ShellState.Unset("ASSIGNED") })

	ExecuteCommand("ASSIGNED=hello", map[int]*jobs.Job{})
	if v, ok := ShellState.GetVar("ASSIGNED"); !ok || v != "hello" {
		t.Errorf("after assignment, ASSIGNED = %q,%v want hello,true", v, ok)
	}
}

func TestAliasExpansion(t *testing.T) {
	ShellState.SetAlias("ll", "ls -la")
	t.Cleanup(func() { ShellState.RemoveAlias("ll") })

	chain, err := ParseString("ll /tmp")
	if err != nil {
		t.Fatal(err)
	}
	expandAliases(chain)
	cmd := chain.Pipelines[0].Commands[0]
	if cmd.Name != "ls" || !reflect.DeepEqual(cmd.Args, []string{"-la", "/tmp"}) {
		t.Errorf("alias expansion = %q %q, want ls [-la /tmp]", cmd.Name, cmd.Args)
	}

	// A quoted command word suppresses alias expansion.
	chain2, _ := ParseString(`"ll" /tmp`)
	expandAliases(chain2)
	if chain2.Pipelines[0].Commands[0].Name != "ll" {
		t.Errorf("quoted alias should not expand, got %q", chain2.Pipelines[0].Commands[0].Name)
	}
}
