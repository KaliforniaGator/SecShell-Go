package cmdmap

import (
	"os"
	"strings"
)

// Phase 3: shell state. Holds the variable store, aliases, and positional
// parameters that the expansion stage (Phase 2) and the new builtins read and
// write. $? remains in lastExitStatus (expand.go), updated by ExecuteProgram.
//
// Variable model (bash-like): exported variables live in the process
// environment so child processes inherit them automatically; non-exported shell
// variables live in vars and are visible only to expansion, not to children.

type shellState struct {
	vars       map[string]string // non-exported shell variables
	aliases    map[string]string // name -> replacement text
	functions  map[string]Node   // name -> function body (a compound command)
	positional []string          // $1, $2, ...
	scriptName string            // $0
}

// ShellState is the process-wide shell state.
var ShellState = &shellState{
	vars:      map[string]string{},
	functions: map[string]Node{},
	// Seed the one alias the old static map carried.
	aliases: map[string]string{"url": "urlencode"},
}

// GetVar resolves a variable: non-exported shell variables take precedence over
// the environment (a plain assignment shadows an inherited env var until
// exported), then the environment.
func (s *shellState) GetVar(name string) (string, bool) {
	if v, ok := s.vars[name]; ok {
		return v, true
	}
	return os.LookupEnv(name)
}

// SetVar sets a shell variable. If the name is already exported (present in the
// environment), the environment copy is updated so children stay in sync;
// otherwise it is stored as a non-exported shell variable.
func (s *shellState) SetVar(name, value string) {
	if _, exported := os.LookupEnv(name); exported {
		_ = os.Setenv(name, value)
		return
	}
	s.vars[name] = value
}

// Export promotes a variable to the environment. With hasValue, it also assigns
// the value; without, an existing shell variable's value is carried over.
func (s *shellState) Export(name, value string, hasValue bool) {
	if !hasValue {
		if v, ok := s.vars[name]; ok {
			value = v
			hasValue = true
		}
	}
	delete(s.vars, name)
	if hasValue {
		_ = os.Setenv(name, value)
	} else if _, ok := os.LookupEnv(name); !ok {
		_ = os.Setenv(name, "")
	}
}

// Unset removes a variable from both the shell store and the environment.
func (s *shellState) Unset(name string) {
	delete(s.vars, name)
	_ = os.Unsetenv(name)
}

// --- aliases ---

func (s *shellState) SetAlias(name, value string) { s.aliases[name] = value }
func (s *shellState) GetAlias(name string) (string, bool) {
	v, ok := s.aliases[name]
	return v, ok
}
func (s *shellState) RemoveAlias(name string) { delete(s.aliases, name) }
func (s *shellState) Aliases() map[string]string { return s.aliases }

// --- positional parameters ---

func (s *shellState) SetPositional(args []string) { s.positional = args }
func (s *shellState) Positional() []string        { return s.positional }

// --- functions ---

func (s *shellState) SetFunc(name string, body Node) { s.functions[name] = body }
func (s *shellState) GetFunc(name string) (Node, bool) {
	b, ok := s.functions[name]
	return b, ok
}

// --- assignment parsing (VAR=value) ---

type assignment struct {
	name  string
	value string
}

// isAssignmentWord reports whether w has the form NAME=... with a valid shell
// variable name before the '='.
func isAssignmentWord(w string) bool {
	eq := strings.IndexByte(w, '=')
	if eq <= 0 {
		return false
	}
	for i := 0; i < eq; i++ {
		c := w[i]
		switch {
		case c == '_', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// splitAssignments separates leading VAR=value words from the command. isPure is
// true when every word is an assignment (no command follows). The returned rest
// CommandNode keeps the original redirections.
func splitAssignments(cmd CommandNode) (assigns []assignment, rest CommandNode, isPure bool) {
	words := append([]string{cmd.Name}, cmd.Args...)
	idx := 0
	for idx < len(words) {
		w := words[idx]
		if w == "" || !isAssignmentWord(w) {
			break
		}
		eq := strings.IndexByte(w, '=')
		assigns = append(assigns, assignment{name: w[:eq], value: w[eq+1:]})
		idx++
	}
	if len(assigns) == 0 {
		return nil, cmd, false
	}
	if idx >= len(words) {
		return assigns, CommandNode{}, true
	}
	rest = CommandNode{
		Name:         words[idx],
		Args:         append([]string{}, words[idx+1:]...),
		StdinFile:    cmd.StdinFile,
		StdoutFile:   cmd.StdoutFile,
		StdoutAppend: cmd.StdoutAppend,
	}
	return assigns, rest, false
}

// applyTempEnv sets assignments in the environment and returns a function that
// restores the previous values — used for prefix assignments that should affect
// only the command they precede.
func applyTempEnv(assigns []assignment) func() {
	type prev struct {
		val string
		had bool
	}
	saved := map[string]prev{}
	for _, a := range assigns {
		if _, ok := saved[a.name]; !ok {
			v, had := os.LookupEnv(a.name)
			saved[a.name] = prev{val: v, had: had}
		}
		_ = os.Setenv(a.name, a.value)
	}
	return func() {
		for name, p := range saved {
			if p.had {
				_ = os.Setenv(name, p.val)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}
}
