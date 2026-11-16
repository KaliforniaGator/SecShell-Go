package cmdmap

import (
	"strconv"
	"strings"
)

// braceExpand implements bash brace expansion: comma lists {a,b,c} and numeric
// ranges {m..n}, with an optional common prefix and suffix, and multiple/nested
// braces via recursion (cartesian product). A string with no valid brace
// expression returns itself unchanged.
func braceExpand(s string) []string {
	// Find the first top-level '{'.
	open := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '{' {
			open = i
			break
		}
	}
	if open == -1 {
		return []string{s}
	}
	// Find the matching '}' for this '{', respecting nesting.
	depth := 0
	close := -1
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				close = i
			}
		}
		if close != -1 {
			break
		}
	}
	if close == -1 {
		return []string{s}
	}

	prefix := s[:open]
	body := s[open+1 : close]
	suffix := s[close+1:]

	options := splitBraceBody(body)
	if options == nil {
		// Not a real brace expression (e.g. {foo} with no comma or range):
		// keep the braces literally but still expand any braces in the suffix.
		rest := braceExpand(suffix)
		out := make([]string, 0, len(rest))
		for _, r := range rest {
			out = append(out, prefix+"{"+body+"}"+r)
		}
		return out
	}

	// Expand the suffix once, then combine with every option (which may itself
	// contain nested braces).
	suffixExpansions := braceExpand(suffix)
	var result []string
	for _, opt := range options {
		for _, optExp := range braceExpand(opt) {
			for _, suf := range suffixExpansions {
				result = append(result, prefix+optExp+suf)
			}
		}
	}
	return result
}

// splitBraceBody returns the brace options for a body, or nil if the body is not
// a valid comma list or numeric range.
func splitBraceBody(body string) []string {
	// Numeric range {m..n}
	if parts := strings.SplitN(body, "..", 2); len(parts) == 2 && !strings.Contains(body, ",") {
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			var out []string
			if start <= end {
				for i := start; i <= end; i++ {
					out = append(out, strconv.Itoa(i))
				}
			} else {
				for i := start; i >= end; i-- {
					out = append(out, strconv.Itoa(i))
				}
			}
			return out
		}
	}

	// Comma list {a,b,c} — split on top-level commas only.
	if !strings.Contains(body, ",") {
		return nil
	}
	var options []string
	depth := 0
	start := 0
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				options = append(options, body[start:i])
				start = i + 1
			}
		}
	}
	options = append(options, body[start:])
	return options
}

// evalArithmetic evaluates a $(( )) integer expression. It first resolves any
// $-expansions, then treats bare identifiers as variables (unset = 0), and
// supports + - * / % , unary minus, and parentheses. On any parse error it
// returns "0", matching bash's behavior of erroring to zero for our purposes.
func evalArithmetic(expr string) string {
	expr = expandString(expr) // resolve $var first
	p := &arithParser{src: expr}
	v := p.parseExpr()
	if p.err || strings.TrimSpace(p.src[p.pos:]) != "" {
		return "0"
	}
	return strconv.Itoa(v)
}

type arithParser struct {
	src string
	pos int
	err bool
}

func (p *arithParser) skipSpaces() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t') {
		p.pos++
	}
}

// parseExpr handles + and -.
func (p *arithParser) parseExpr() int {
	v := p.parseTerm()
	for {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			break
		}
		op := p.src[p.pos]
		if op != '+' && op != '-' {
			break
		}
		p.pos++
		r := p.parseTerm()
		if op == '+' {
			v += r
		} else {
			v -= r
		}
	}
	return v
}

// parseTerm handles * / %.
func (p *arithParser) parseTerm() int {
	v := p.parseFactor()
	for {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			break
		}
		op := p.src[p.pos]
		if op != '*' && op != '/' && op != '%' {
			break
		}
		p.pos++
		r := p.parseFactor()
		switch op {
		case '*':
			v *= r
		case '/':
			if r == 0 {
				p.err = true
				return 0
			}
			v /= r
		case '%':
			if r == 0 {
				p.err = true
				return 0
			}
			v %= r
		}
	}
	return v
}

// parseFactor handles numbers, identifiers, parentheses, and unary minus/plus.
func (p *arithParser) parseFactor() int {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		p.err = true
		return 0
	}
	c := p.src[p.pos]
	switch {
	case c == '(':
		p.pos++
		v := p.parseExpr()
		p.skipSpaces()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			p.err = true
			return 0
		}
		p.pos++
		return v
	case c == '-':
		p.pos++
		return -p.parseFactor()
	case c == '+':
		p.pos++
		return p.parseFactor()
	case c >= '0' && c <= '9':
		start := p.pos
		for p.pos < len(p.src) && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
			p.pos++
		}
		n, err := strconv.Atoi(p.src[start:p.pos])
		if err != nil {
			p.err = true
			return 0
		}
		return n
	case c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
		start := p.pos
		for p.pos < len(p.src) {
			ch := p.src[p.pos]
			if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
				p.pos++
			} else {
				break
			}
		}
		name := p.src[start:p.pos]
		val, _ := lookupVar(name)
		n, _ := strconv.Atoi(strings.TrimSpace(val)) // unset/non-numeric -> 0
		return n
	default:
		p.err = true
		return 0
	}
}
