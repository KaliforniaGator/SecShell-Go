package cmdmap

import "testing"

func TestIsIncompleteCommand(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"ls -la", false},
		{"echo hi; pwd", false},
		{"if true; then echo hi; fi", false},
		{"if true; then echo hi", true},   // missing fi
		{"for i in a b; do echo $i", true}, // missing done
		{"while true; do :", true},         // missing done
		{"case $x in a) echo a;;", true},   // missing esac
		{"(echo hi", true},                 // unbalanced (
		{"(echo hi)", false},
		{"{ echo hi; }", false},
		{"{ echo hi;", true}, // missing }
		{`echo "unterminated`, true},
		{`echo 'unterminated`, true},
		{`echo "closed"`, false},
		{"echo foo |", true},    // trailing pipe
		{"echo foo &&", true},   // trailing &&
		{"echo foo \\", true},   // trailing backslash
		{"echo foo", false},
		{"nested if: if a; then if b; then c; fi", true}, // one fi missing
	}
	for _, c := range cases {
		if got := IsIncompleteCommand(c.input); got != c.want {
			t.Errorf("IsIncompleteCommand(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
