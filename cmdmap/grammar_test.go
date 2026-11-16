package cmdmap

import (
	"reflect"
	"testing"
)

// --- lexer additions ---

func TestLexNewOperators(t *testing.T) {
	cases := []struct {
		input string
		want  []TokenType
	}{
		{"a; b", []TokenType{TokenWord, TokenSemi, TokenWord}},
		{"a;;b", []TokenType{TokenWord, TokenDSemi, TokenWord}},
		{"(a)", []TokenType{TokenLParen, TokenWord, TokenRParen}},
		{"a &> f", []TokenType{TokenWord, TokenRedirectBoth, TokenWord}},
		{"a <<< x", []TokenType{TokenWord, TokenHereString, TokenWord}},
		{"a\nb", []TokenType{TokenWord, TokenSemi, TokenWord}}, // newline separates
	}
	for _, c := range cases {
		var got []TokenType
		for _, tk := range Tokenize(c.input) {
			got = append(got, tk.Type)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Tokenize(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestLexFileDescriptorRedirect(t *testing.T) {
	toks := Tokenize("cmd 2>err")
	// Expect: WORD cmd, REDIRECT_OUT(fd=2), WORD err  — the 2 must attach to >.
	if len(toks) != 3 {
		t.Fatalf("got %d tokens, want 3: %+v", len(toks), toks)
	}
	if toks[1].Type != TokenRedirectOut || toks[1].Fd != 2 {
		t.Errorf("token1 = %+v, want REDIRECT_OUT fd=2", toks[1])
	}
	// A bare > carries fd -1 (unspecified).
	if toks := Tokenize("cmd >out"); toks[1].Fd != -1 {
		t.Errorf("bare > fd = %d, want -1", toks[1].Fd)
	}
}

// --- existing Parse path: ; and redirections ---

func TestParseSemicolonSequence(t *testing.T) {
	chain, err := ParseString("a; b; c")
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}
	if len(chain.Pipelines) != 3 {
		t.Fatalf("got %d pipelines, want 3", len(chain.Pipelines))
	}
	for i, op := range chain.Operators {
		if op != OpSeq {
			t.Errorf("operator %d = %q, want OpSeq", i, op)
		}
	}
	// Trailing ; must not leave a dangling operator.
	chain2, _ := ParseString("ls ;")
	if len(chain2.Operators) != len(chain2.Pipelines)-1 {
		t.Errorf("trailing ;: operators=%d pipelines=%d", len(chain2.Operators), len(chain2.Pipelines))
	}
}

func TestParseRedirectionList(t *testing.T) {
	chain, err := ParseString("cmd >out 2>err")
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}
	cmd := chain.Pipelines[0].Commands[0]
	// Legacy stdout field set by >out; stderr only in the Redirs list.
	if cmd.StdoutFile != "out" {
		t.Errorf("StdoutFile = %q, want out", cmd.StdoutFile)
	}
	if len(cmd.Redirs) != 2 {
		t.Fatalf("got %d redirs, want 2: %+v", len(cmd.Redirs), cmd.Redirs)
	}
	errRedir := cmd.Redirs[1]
	if errRedir.Op != RedirOut || errRedir.Fd != 2 || errRedir.Target != "err" {
		t.Errorf("stderr redir = %+v, want RedirOut fd=2 target=err", errRedir)
	}
}

func TestParseHereStringAndBoth(t *testing.T) {
	chain, _ := ParseString(`cmd <<< "hi there"`)
	r := chain.Pipelines[0].Commands[0].Redirs
	if len(r) != 1 || r[0].Op != RedirHereStr || r[0].Target != "hi there" {
		t.Errorf("here-string redir = %+v", r)
	}
	chain2, _ := ParseString("cmd &> all")
	r2 := chain2.Pipelines[0].Commands[0].Redirs
	if len(r2) != 1 || r2[0].Op != RedirBoth || r2[0].Target != "all" {
		t.Errorf("&> redir = %+v", r2)
	}
}

// --- compound grammar (ParseProgram) ---

func TestParseProgramIf(t *testing.T) {
	prog, err := ParseProgram("if true; then echo yes; else echo no; fi")
	if err != nil {
		t.Fatalf("ParseProgram: %v", err)
	}
	if len(prog.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(prog.Items))
	}
	ifNode, ok := prog.Items[0].AndOr.Pipelines[0].Cmds[0].(*IfNode)
	if !ok {
		t.Fatalf("not an IfNode: %T", prog.Items[0].AndOr.Pipelines[0].Cmds[0])
	}
	if ifNode.Cond == nil || ifNode.Then == nil || ifNode.Else == nil {
		t.Errorf("if parts missing: %+v", ifNode)
	}
	// The 'then' branch is `echo yes`.
	thenCmd := ifNode.Then.Items[0].AndOr.Pipelines[0].Cmds[0].(*SimpleCmd)
	if thenCmd.Name != "echo" || !reflect.DeepEqual(thenCmd.Args, []string{"yes"}) {
		t.Errorf("then branch = %q %q", thenCmd.Name, thenCmd.Args)
	}
}

func TestParseProgramFor(t *testing.T) {
	prog, err := ParseProgram("for i in a b c; do echo $i; done")
	if err != nil {
		t.Fatalf("ParseProgram: %v", err)
	}
	forNode := prog.Items[0].AndOr.Pipelines[0].Cmds[0].(*ForNode)
	if forNode.Var != "i" {
		t.Errorf("for var = %q, want i", forNode.Var)
	}
	if len(forNode.Words) != 3 {
		t.Errorf("for words = %+v, want 3", forNode.Words)
	}
	body := forNode.Body.Items[0].AndOr.Pipelines[0].Cmds[0].(*SimpleCmd)
	if body.Name != "echo" {
		t.Errorf("for body = %q, want echo", body.Name)
	}
}

func TestParseProgramWhileCaseFuncSubshell(t *testing.T) {
	cases := []struct {
		name  string
		input string
		check func(t *testing.T, n Node)
	}{
		{"while", "while true; do echo x; done", func(t *testing.T, n Node) {
			if w, ok := n.(*WhileNode); !ok || w.Until {
				t.Errorf("want while node, got %T", n)
			}
		}},
		{"until", "until false; do echo x; done", func(t *testing.T, n Node) {
			if w, ok := n.(*WhileNode); !ok || !w.Until {
				t.Errorf("want until node, got %T", n)
			}
		}},
		{"case", "case $x in a) echo a;; b|c) echo bc;; esac", func(t *testing.T, n Node) {
			c, ok := n.(*CaseNode)
			if !ok {
				t.Fatalf("want case node, got %T", n)
			}
			if len(c.Items) != 2 || len(c.Items[1].Patterns) != 2 {
				t.Errorf("case items = %+v", c.Items)
			}
		}},
		{"funcdef", "greet() { echo hi; }", func(t *testing.T, n Node) {
			f, ok := n.(*FuncNode)
			if !ok {
				t.Fatalf("want func node, got %T", n)
			}
			if f.Name != "greet" {
				t.Errorf("func name = %q", f.Name)
			}
			if _, ok := f.Body.(*GroupNode); !ok {
				t.Errorf("func body = %T, want group", f.Body)
			}
		}},
		{"subshell", "(cd /tmp; ls)", func(t *testing.T, n Node) {
			s, ok := n.(*SubshellNode)
			if !ok {
				t.Fatalf("want subshell, got %T", n)
			}
			if len(s.Body.Items) != 2 {
				t.Errorf("subshell items = %d, want 2", len(s.Body.Items))
			}
		}},
		{"group", "{ echo a; echo b; }", func(t *testing.T, n Node) {
			if _, ok := n.(*GroupNode); !ok {
				t.Fatalf("want group, got %T", n)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prog, err := ParseProgram(c.input)
			if err != nil {
				t.Fatalf("ParseProgram(%q): %v", c.input, err)
			}
			c.check(t, prog.Items[0].AndOr.Pipelines[0].Cmds[0])
		})
	}
}

func TestParseProgramReservedWordAsArg(t *testing.T) {
	// `done` as a non-initial word is an ordinary argument, not a terminator.
	prog, err := ParseProgram("echo done")
	if err != nil {
		t.Fatalf("ParseProgram: %v", err)
	}
	cmd := prog.Items[0].AndOr.Pipelines[0].Cmds[0].(*SimpleCmd)
	if cmd.Name != "echo" || !reflect.DeepEqual(cmd.Args, []string{"done"}) {
		t.Errorf("got %q %q, want echo [done]", cmd.Name, cmd.Args)
	}
}

func TestParseProgramPipelinesAndAndOr(t *testing.T) {
	prog, err := ParseProgram("a | b && c || d")
	if err != nil {
		t.Fatalf("ParseProgram: %v", err)
	}
	ao := prog.Items[0].AndOr
	if len(ao.Pipelines) != 3 {
		t.Fatalf("pipelines = %d, want 3", len(ao.Pipelines))
	}
	if len(ao.Pipelines[0].Cmds) != 2 {
		t.Errorf("first pipeline cmds = %d, want 2", len(ao.Pipelines[0].Cmds))
	}
	if !reflect.DeepEqual(ao.Ops, []ChainOperator{OpAnd, OpOr}) {
		t.Errorf("ops = %v", ao.Ops)
	}
}
