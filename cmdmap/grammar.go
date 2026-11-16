package cmdmap

import "fmt"

// Phase 4: a recursive-descent parser that builds the full program AST (ast.go),
// including compound commands. It reuses the quote-aware lexer (Tokenize). It is
// not yet wired into execution — Phase 5 adds the executor and the multi-line
// reader and switches ExecuteCommand over to it. The existing ParseString/Parse
// path continues to run the live shell until then.

// listStopWords are reserved words that terminate a command list when they
// appear in command position (i.e. as the first word of a would-be command).
// As a non-initial word they are ordinary arguments, which parseSimpleCommand
// handles by only consulting this set for the command list boundary.
var listStopWords = map[string]bool{
	"then": true, "else": true, "elif": true, "fi": true,
	"do": true, "done": true, "esac": true, "in": true, "}": true,
}

type progParser struct {
	toks []Token
	pos  int
}

// ParseProgram tokenizes and parses input into a program AST.
func ParseProgram(input string) (*ListNode, error) {
	p := &progParser{toks: Tokenize(input)}
	list, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() {
		return nil, fmt.Errorf("unexpected token %q", p.cur().Value)
	}
	if list == nil || len(list.Items) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return list, nil
}

// --- token navigation ---

func (p *progParser) atEnd() bool { return p.pos >= len(p.toks) }

func (p *progParser) cur() Token {
	if p.atEnd() {
		return Token{Type: "EOF"}
	}
	return p.toks[p.pos]
}

func (p *progParser) peekAhead(n int) Token {
	if p.pos+n >= len(p.toks) {
		return Token{Type: "EOF"}
	}
	return p.toks[p.pos+n]
}

func (p *progParser) peekType() TokenType { return p.cur().Type }
func (p *progParser) advance() Token      { t := p.cur(); p.pos++; return t }
func (p *progParser) isWord(v string) bool {
	t := p.cur()
	return t.Type == TokenWord && t.Value == v
}

func (p *progParser) expectWord(v string) error {
	if p.isWord(v) {
		p.advance()
		return nil
	}
	return fmt.Errorf("expected %q, got %q", v, p.cur().Value)
}

// skipSeparators consumes ; and newline separators (newlines lex as TokenSemi).
func (p *progParser) skipSeparators() {
	for p.peekType() == TokenSemi {
		p.advance()
	}
}

// --- grammar ---

// atListEnd reports whether the current position ends the enclosing list.
func (p *progParser) atListEnd() bool {
	if p.atEnd() {
		return true
	}
	t := p.cur()
	if t.Type == TokenRParen || t.Type == TokenDSemi {
		return true
	}
	if t.Type == TokenWord && listStopWords[t.Value] {
		return true
	}
	return false
}

func (p *progParser) parseList() (*ListNode, error) {
	list := &ListNode{}
	for {
		p.skipSeparators()
		if p.atListEnd() {
			break
		}
		andor, err := p.parseAndOr()
		if err != nil {
			return nil, err
		}
		item := AndOrItem{AndOr: andor}
		if p.peekType() == TokenBackground {
			item.Background = true
			p.advance()
		} else if p.peekType() == TokenSemi {
			p.advance()
		}
		list.Items = append(list.Items, item)
	}
	return list, nil
}

func (p *progParser) parseAndOr() (*AndOrNode, error) {
	first, err := p.parsePipeline()
	if err != nil {
		return nil, err
	}
	ao := &AndOrNode{Pipelines: []*PipeNode{first}}
	for p.peekType() == TokenAnd || p.peekType() == TokenOr {
		op := OpAnd
		if p.peekType() == TokenOr {
			op = OpOr
		}
		p.advance()
		p.skipSeparators() // a newline may follow && or ||
		pl, err := p.parsePipeline()
		if err != nil {
			return nil, err
		}
		ao.Ops = append(ao.Ops, op)
		ao.Pipelines = append(ao.Pipelines, pl)
	}
	return ao, nil
}

func (p *progParser) parsePipeline() (*PipeNode, error) {
	cmd, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	pipe := &PipeNode{Cmds: []Node{cmd}}
	for p.peekType() == TokenPipe {
		p.advance()
		p.skipSeparators()
		c, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		pipe.Cmds = append(pipe.Cmds, c)
	}
	return pipe, nil
}

func (p *progParser) parseCommand() (Node, error) {
	t := p.cur()
	if t.Type == TokenWord {
		switch t.Value {
		case "if":
			return p.parseIf()
		case "for":
			return p.parseFor()
		case "while":
			return p.parseWhile(false)
		case "until":
			return p.parseWhile(true)
		case "case":
			return p.parseCase()
		case "{":
			return p.parseGroup()
		case "function":
			return p.parseFunctionKeyword()
		}
		// Function definition:  name ( )
		if isName(t.Value) && p.peekAhead(1).Type == TokenLParen {
			return p.parseFuncDef()
		}
	}
	if t.Type == TokenLParen {
		return p.parseSubshell()
	}
	return p.parseSimpleCommand()
}

func (p *progParser) parseSimpleCommand() (Node, error) {
	sc := &SimpleCmd{}
	for {
		t := p.cur()
		if t.Type == TokenWord {
			if sc.Name == "" && sc.NameWord == nil {
				sc.Name = t.Value
				sc.NameWord = t.Parts
			} else {
				sc.Args = append(sc.Args, t.Value)
				sc.ArgWords = append(sc.ArgWords, t.Parts)
			}
			p.advance()
			continue
		}
		r, ok, err := p.tryParseRedir()
		if err != nil {
			return nil, err
		}
		if ok {
			sc.Redirs = append(sc.Redirs, *r)
			continue
		}
		break
	}
	if sc.Name == "" && len(sc.Redirs) == 0 {
		return nil, fmt.Errorf("expected command, got %q", p.cur().Value)
	}
	return sc, nil
}

func (p *progParser) tryParseRedir() (*Redirection, bool, error) {
	t := p.cur()
	var op RedirOp
	switch t.Type {
	case TokenRedirectIn:
		op = RedirIn
	case TokenRedirectOut:
		op = RedirOut
	case TokenRedirectAppend:
		op = RedirAppend
	case TokenRedirectBoth:
		op = RedirBoth
	case TokenHereString:
		op = RedirHereStr
	case TokenRedirectHereDoc:
		op = RedirHeredoc
	default:
		return nil, false, nil
	}
	fd := t.Fd
	p.advance()
	if p.peekType() != TokenWord {
		return nil, false, fmt.Errorf("expected redirection target after %q", t.Value)
	}
	target := p.advance()
	return &Redirection{Fd: fd, Op: op, Target: target.Value}, true, nil
}

func (p *progParser) collectRedirs() []Redirection {
	var redirs []Redirection
	for {
		r, ok, err := p.tryParseRedir()
		if err != nil || !ok {
			break
		}
		redirs = append(redirs, *r)
	}
	return redirs
}

func (p *progParser) parseIf() (Node, error) {
	p.advance() // if
	cond, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if err := p.expectWord("then"); err != nil {
		return nil, err
	}
	then, err := p.parseList()
	if err != nil {
		return nil, err
	}
	node := &IfNode{Cond: cond, Then: then}
	for p.isWord("elif") {
		p.advance()
		c, err := p.parseList()
		if err != nil {
			return nil, err
		}
		if err := p.expectWord("then"); err != nil {
			return nil, err
		}
		th, err := p.parseList()
		if err != nil {
			return nil, err
		}
		node.Elifs = append(node.Elifs, ElifPart{Cond: c, Then: th})
	}
	if p.isWord("else") {
		p.advance()
		el, err := p.parseList()
		if err != nil {
			return nil, err
		}
		node.Else = el
	}
	if err := p.expectWord("fi"); err != nil {
		return nil, err
	}
	node.Redirs = p.collectRedirs()
	return node, nil
}

func (p *progParser) parseFor() (Node, error) {
	p.advance() // for
	if p.peekType() != TokenWord {
		return nil, fmt.Errorf("expected variable name after 'for'")
	}
	node := &ForNode{Var: p.advance().Value}
	if p.isWord("in") {
		p.advance()
		for p.peekType() == TokenWord && !listStopWords[p.cur().Value] {
			w := p.advance()
			node.Words = append(node.Words, WordSpec{Value: w.Value, Parts: w.Parts})
		}
	}
	p.skipSeparators()
	if err := p.expectWord("do"); err != nil {
		return nil, err
	}
	body, err := p.parseList()
	if err != nil {
		return nil, err
	}
	node.Body = body
	if err := p.expectWord("done"); err != nil {
		return nil, err
	}
	node.Redirs = p.collectRedirs()
	return node, nil
}

func (p *progParser) parseWhile(until bool) (Node, error) {
	p.advance() // while / until
	cond, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if err := p.expectWord("do"); err != nil {
		return nil, err
	}
	body, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if err := p.expectWord("done"); err != nil {
		return nil, err
	}
	node := &WhileNode{Cond: cond, Body: body, Until: until}
	node.Redirs = p.collectRedirs()
	return node, nil
}

func (p *progParser) parseCase() (Node, error) {
	p.advance() // case
	if p.peekType() != TokenWord {
		return nil, fmt.Errorf("expected word after 'case'")
	}
	w := p.advance()
	node := &CaseNode{Word: WordSpec{Value: w.Value, Parts: w.Parts}}
	if err := p.expectWord("in"); err != nil {
		return nil, err
	}
	p.skipSeparators()
	for !p.isWord("esac") && !p.atEnd() {
		if p.peekType() == TokenLParen {
			p.advance() // optional leading (
		}
		var pats []WordSpec
		for {
			if p.peekType() != TokenWord {
				return nil, fmt.Errorf("expected pattern in case, got %q", p.cur().Value)
			}
			pw := p.advance()
			pats = append(pats, WordSpec{Value: pw.Value, Parts: pw.Parts})
			if p.peekType() == TokenPipe {
				p.advance()
				continue
			}
			break
		}
		if p.peekType() != TokenRParen {
			return nil, fmt.Errorf("expected ')' after case pattern")
		}
		p.advance()
		body, err := p.parseList()
		if err != nil {
			return nil, err
		}
		node.Items = append(node.Items, CaseItem{Patterns: pats, Body: body})
		if p.peekType() == TokenDSemi {
			p.advance()
		}
		p.skipSeparators()
	}
	if err := p.expectWord("esac"); err != nil {
		return nil, err
	}
	node.Redirs = p.collectRedirs()
	return node, nil
}

func (p *progParser) parseSubshell() (Node, error) {
	p.advance() // (
	body, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if p.peekType() != TokenRParen {
		return nil, fmt.Errorf("expected ')' to close subshell")
	}
	p.advance()
	return &SubshellNode{Body: body, Redirs: p.collectRedirs()}, nil
}

func (p *progParser) parseGroup() (Node, error) {
	p.advance() // {
	body, err := p.parseList()
	if err != nil {
		return nil, err
	}
	if err := p.expectWord("}"); err != nil {
		return nil, err
	}
	return &GroupNode{Body: body, Redirs: p.collectRedirs()}, nil
}

func (p *progParser) parseFuncDef() (Node, error) {
	name := p.advance().Value // name
	p.advance()               // (
	if p.peekType() != TokenRParen {
		return nil, fmt.Errorf("expected ')' in function definition")
	}
	p.advance() // )
	p.skipSeparators()
	body, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	return &FuncNode{Name: name, Body: body}, nil
}

func (p *progParser) parseFunctionKeyword() (Node, error) {
	p.advance() // function
	if p.peekType() != TokenWord {
		return nil, fmt.Errorf("expected name after 'function'")
	}
	name := p.advance().Value
	if p.peekType() == TokenLParen {
		p.advance()
		if p.peekType() != TokenRParen {
			return nil, fmt.Errorf("expected ')' in function definition")
		}
		p.advance()
	}
	p.skipSeparators()
	body, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	return &FuncNode{Name: name, Body: body}, nil
}

// isName reports whether s is a valid function/variable identifier.
func isName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}
