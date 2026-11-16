package cmdmap

import (
	"reflect"
	"testing"
)

// wordValues extracts the flattened Value of every WORD token, ignoring
// operators. This is the behavior the rest of the shell relied on before the
// quote-aware lexer existed, so these cases guard against regressions.
func wordValues(tokens []Token) []string {
	var out []string
	for _, t := range tokens {
		if t.Type == TokenWord {
			out = append(out, t.Value)
		}
	}
	return out
}

func TestTokenizeFlattenedValues(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain", "ls -la /tmp", []string{"ls", "-la", "/tmp"}},
		{"double quotes join spaces", `echo "hello world"`, []string{"echo", "hello world"}},
		{"single quotes join spaces", `echo 'hello world'`, []string{"echo", "hello world"}},
		{"adjacent quotes concatenate", `echo a"b"'c'`, []string{"echo", "abc"}},
		{"backslash escapes space", `echo a\ b`, []string{"echo", "a b"}},
		{"backslash escapes dollar", `echo \$HOME`, []string{"echo", "$HOME"}},
		{"nested different quote", `echo "it's"`, []string{"echo", "it's"}},
		{"double inside single literal", `echo 'a"b'`, []string{"echo", `a"b`}},
		{"tabs split", "a\tb", []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := wordValues(Tokenize(c.input))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Tokenize(%q) words = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestTokenizeOperators(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []TokenType
	}{
		{"pipe", "a | b", []TokenType{TokenWord, TokenPipe, TokenWord}},
		{"or", "a || b", []TokenType{TokenWord, TokenOr, TokenWord}},
		{"and", "a && b", []TokenType{TokenWord, TokenAnd, TokenWord}},
		{"background", "a &", []TokenType{TokenWord, TokenBackground}},
		{"redirect out", "a > f", []TokenType{TokenWord, TokenRedirectOut, TokenWord}},
		{"redirect append", "a >> f", []TokenType{TokenWord, TokenRedirectAppend, TokenWord}},
		{"redirect in", "a < f", []TokenType{TokenWord, TokenRedirectIn, TokenWord}},
		{"operator chars quoted are literal", `echo "a | b"`, []TokenType{TokenWord, TokenWord}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			toks := Tokenize(c.input)
			var got []TokenType
			for _, tk := range toks {
				got = append(got, tk.Type)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Tokenize(%q) types = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

// TestTokenizeQuoteParts is the new capability: each WORD token records how each
// span was quoted so a later expansion stage can treat $VAR, "$VAR" and '$VAR'
// differently.
func TestTokenizeQuoteParts(t *testing.T) {
	cases := []struct {
		name  string
		input string // a single word
		want  []WordPart
	}{
		{"unquoted", "$HOME", []WordPart{{Text: "$HOME", Quote: QuoteNone}}},
		{"single quoted suppresses", "'$HOME'", []WordPart{{Text: "$HOME", Quote: QuoteSingle}}},
		{"double quoted expands", `"$HOME"`, []WordPart{{Text: "$HOME", Quote: QuoteDouble}}},
		{"mixed spans", `a"$b"'$c'`, []WordPart{
			{Text: "a", Quote: QuoteNone},
			{Text: "$b", Quote: QuoteDouble},
			{Text: "$c", Quote: QuoteSingle},
		}},
		{"escaped dollar is literal", `\$x`, []WordPart{
			{Text: "$", Quote: QuoteSingle},
			{Text: "x", Quote: QuoteNone},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			toks := Tokenize(c.input)
			if len(toks) != 1 || toks[0].Type != TokenWord {
				t.Fatalf("Tokenize(%q) = %+v, want a single WORD token", c.input, toks)
			}
			if !reflect.DeepEqual(toks[0].Parts, c.want) {
				t.Errorf("Tokenize(%q) parts = %+v, want %+v", c.input, toks[0].Parts, c.want)
			}
			// Invariant: concatenating part text must equal the flattened Value.
			var joined string
			for _, p := range toks[0].Parts {
				joined += p.Text
			}
			if joined != toks[0].Value {
				t.Errorf("parts concat %q != Value %q", joined, toks[0].Value)
			}
		})
	}
}

func TestParseRedirectionAndPipeline(t *testing.T) {
	chain, err := ParseString("cat < in.txt | grep foo >> out.txt")
	if err != nil {
		t.Fatalf("ParseString error: %v", err)
	}
	if len(chain.Pipelines) != 1 {
		t.Fatalf("got %d pipelines, want 1", len(chain.Pipelines))
	}
	cmds := chain.Pipelines[0].Commands
	if len(cmds) != 2 {
		t.Fatalf("got %d commands, want 2", len(cmds))
	}
	if cmds[0].Name != "cat" || cmds[0].StdinFile != "in.txt" {
		t.Errorf("cmd0 = %+v, want cat with stdin in.txt", cmds[0])
	}
	if cmds[1].Name != "grep" || cmds[1].StdoutFile != "out.txt" || !cmds[1].StdoutAppend {
		t.Errorf("cmd1 = %+v, want grep appending to out.txt", cmds[1])
	}
}

func TestParseChainOperators(t *testing.T) {
	chain, err := ParseString("a && b || c")
	if err != nil {
		t.Fatalf("ParseString error: %v", err)
	}
	if len(chain.Pipelines) != 3 {
		t.Fatalf("got %d pipelines, want 3", len(chain.Pipelines))
	}
	want := []ChainOperator{OpAnd, OpOr}
	if !reflect.DeepEqual(chain.Operators, want) {
		t.Errorf("operators = %v, want %v", chain.Operators, want)
	}
}

// TestParseArgWordsParallel verifies the quote-aware ArgWords stay in lockstep
// with the flattened Args, which is the contract the expansion stage relies on.
func TestParseArgWordsParallel(t *testing.T) {
	chain, err := ParseString(`echo plain "$x" '$y'`)
	if err != nil {
		t.Fatalf("ParseString error: %v", err)
	}
	cmd := chain.Pipelines[0].Commands[0]
	if cmd.Name != "echo" {
		t.Fatalf("name = %q, want echo", cmd.Name)
	}
	if len(cmd.Args) != len(cmd.ArgWords) {
		t.Fatalf("Args len %d != ArgWords len %d", len(cmd.Args), len(cmd.ArgWords))
	}
	if cmd.ArgWords[1][0].Quote != QuoteDouble {
		t.Errorf(`arg "$x" quote = %v, want QuoteDouble`, cmd.ArgWords[1][0].Quote)
	}
	if cmd.ArgWords[2][0].Quote != QuoteSingle {
		t.Errorf(`arg '$y' quote = %v, want QuoteSingle`, cmd.ArgWords[2][0].Quote)
	}
}
