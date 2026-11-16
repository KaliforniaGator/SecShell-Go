package cmdmap

// IsIncompleteCommand reports whether input is an unfinished command that should
// prompt for a continuation line (PS2) rather than execute. It detects:
//   - an unclosed single or double quote
//   - a trailing backslash line-continuation
//   - a line ending in a pipe or && / || operator
//   - an unbalanced compound construct: if/for/while/until/case openers without
//     their fi/done/esac, unbalanced ( ) or { }
//
// It is a heuristic sufficient for the interactive reader; the parser remains the
// source of truth once a complete command is assembled.
func IsIncompleteCommand(input string) bool {
	if hasOpenQuote(input) {
		return true
	}
	if hasTrailingBackslash(input) {
		return true
	}

	toks := Tokenize(input)
	// Drop trailing separator tokens so "for i in a; " still counts its openers.
	depth := 0
	openers := map[string]int{"if": 1, "for": 1, "while": 1, "until": 1, "case": 1}
	closers := map[string]int{"fi": 1, "done": 1, "esac": 1}
	parens := 0
	braces := 0

	var lastSignificant Token
	for _, t := range toks {
		switch t.Type {
		case TokenWord:
			switch {
			case openers[t.Value] == 1:
				depth++
			case closers[t.Value] == 1:
				depth--
			case t.Value == "{":
				braces++
			case t.Value == "}":
				braces--
			}
		case TokenLParen:
			parens++
		case TokenRParen:
			parens--
		}
		if t.Type != TokenSemi {
			lastSignificant = t
		}
	}

	if depth > 0 || parens > 0 || braces > 0 {
		return true
	}

	// A line ending with a connector expects more.
	switch lastSignificant.Type {
	case TokenPipe, TokenAnd, TokenOr:
		return true
	}
	return false
}

// hasOpenQuote reports whether input ends inside a single or double quote,
// honoring backslash escaping.
func hasOpenQuote(input string) bool {
	inQuote := false
	var quote rune
	for i := 0; i < len(input); i++ {
		c := rune(input[i])
		if c == '\\' && !inQuote {
			i++ // skip the escaped character
			continue
		}
		if c == '\\' && inQuote && quote == '"' {
			i++
			continue
		}
		if inQuote {
			if c == quote {
				inQuote = false
			}
			continue
		}
		if c == '\'' || c == '"' {
			inQuote = true
			quote = c
		}
	}
	return inQuote
}

// hasTrailingBackslash reports whether input ends with an unescaped backslash.
func hasTrailingBackslash(input string) bool {
	n := 0
	for i := len(input) - 1; i >= 0 && input[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}
