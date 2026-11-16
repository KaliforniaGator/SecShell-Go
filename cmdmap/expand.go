package cmdmap

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Phase 2: the expansion stage. It sits between parse and execute and rewrites
// each CommandNode's Name/Args/redirection targets by applying bash-style
// expansions in bash order, honoring the per-span quoting captured by the
// quote-aware lexer (Phase 1):
//
//	brace -> tilde -> parameter -> command substitution -> arithmetic
//	      -> word splitting -> pathname (glob)
//
// Variable state here is intentionally minimal (environment + a few specials);
// Phase 3 replaces lookupVar with a real shell-variable store.

// lastExitStatus backs $? for parameter expansion. executeChain updates it after
// each command line. It is a package-level stand-in until Phase 3's shell state.
var lastExitStatus int

// lookupVar resolves a variable name for expansion. Special parameters are
// handled first, then positional parameters, then the shell state (shell vars +
// environment).
func lookupVar(name string) (string, bool) {
	switch name {
	case "?":
		return strconv.Itoa(lastExitStatus), true
	case "$":
		return strconv.Itoa(os.Getpid()), true
	case "#":
		return strconv.Itoa(len(ShellState.positional)), true
	case "@", "*":
		return strings.Join(ShellState.positional, " "), true
	case "0":
		if ShellState.scriptName != "" {
			return ShellState.scriptName, true
		}
		return "secshell", true
	}
	// $1..$9 positional parameters.
	if len(name) == 1 && name[0] >= '1' && name[0] <= '9' {
		idx := int(name[0]-'0') - 1
		if idx < len(ShellState.positional) {
			return ShellState.positional[idx], true
		}
		return "", false
	}
	return ShellState.GetVar(name)
}

// segment is a run of expanded text tagged with whether it is subject to word
// splitting and globbing. Unquoted expansion output is splittable/globbable;
// quoted (single or double) text is not.
type segment struct {
	text   string
	quoted bool
}

// ExpandChain rewrites every command node in the chain in place. It is the single
// wiring point called from ExecuteCommand after parsing succeeds.
func ExpandChain(chain *ChainNode) {
	for pi := range chain.Pipelines {
		for ci := range chain.Pipelines[pi].Commands {
			expandCommandNode(&chain.Pipelines[pi].Commands[ci])
		}
	}
}

func expandCommandNode(cmd *CommandNode) {
	// Expand the command name; if it splits/globs into several words, the first is
	// the command and the rest are prepended to the arguments (bash semantics).
	var out []string
	nameWords := expandWord(cmd.NameWord, cmd.Name)
	out = append(out, nameWords...)

	for i, argFlat := range cmd.Args {
		var parts []WordPart
		if i < len(cmd.ArgWords) {
			parts = cmd.ArgWords[i]
		}
		out = append(out, expandWord(parts, argFlat)...)
	}

	if len(out) == 0 {
		// Everything expanded away (e.g. an unset unquoted variable). Leave the
		// node effectively empty; callers already guard on an empty Name.
		cmd.Name = ""
		cmd.Args = nil
	} else {
		cmd.Name = out[0]
		cmd.Args = out[1:]
	}
	// Keep the quote-aware fields from being reused after expansion.
	cmd.NameWord = nil
	cmd.ArgWords = nil

	// Redirection targets are single words; expand them simply (tilde + params).
	cmd.StdinFile = expandRedirTarget(cmd.StdinFile)
	cmd.StdoutFile = expandRedirTarget(cmd.StdoutFile)
	for i := range cmd.Redirs {
		// A here-string is a word: expand it like an argument (quote-aware). Other
		// targets are filenames: tilde + parameter expansion only.
		if cmd.Redirs[i].Op == RedirHereStr {
			cmd.Redirs[i].Target = expandString(expandTilde(cmd.Redirs[i].Target))
		} else {
			cmd.Redirs[i].Target = expandRedirTarget(cmd.Redirs[i].Target)
		}
	}
}

// expandWord applies the full expansion pipeline to one word and returns the
// resulting words (0..n). flat is the quote-stripped literal used as a fallback
// when parts is nil.
func expandWord(parts []WordPart, flat string) []string {
	if len(parts) == 0 {
		if flat == "" {
			return nil
		}
		parts = []WordPart{{Text: flat, Quote: QuoteNone}}
	}

	// Brace expansion happens first, and only when the whole word is unquoted.
	variants := [][]WordPart{parts}
	if allUnquoted(parts) {
		expanded := braceExpand(flat)
		if len(expanded) > 1 {
			variants = variants[:0]
			for _, v := range expanded {
				variants = append(variants, []WordPart{{Text: v, Quote: QuoteNone}})
			}
		}
	}

	var result []string
	for _, variant := range variants {
		segs := expandParts(variant)
		for _, word := range splitIntoWords(segs) {
			result = append(result, globWord(word)...)
		}
	}
	return result
}

// expandToString expands a word to a single string with parameter, command, and
// arithmetic expansion but no word-splitting or globbing. Used where a single
// value is required: the case word, case patterns, and for-loop fallback.
func expandToString(parts []WordPart, flat string) string {
	if len(parts) == 0 {
		parts = []WordPart{{Text: flat, Quote: QuoteNone}}
	}
	var b strings.Builder
	for _, s := range expandParts(parts) {
		b.WriteString(s.text)
	}
	return b.String()
}

func allUnquoted(parts []WordPart) bool {
	for _, p := range parts {
		if p.Quote != QuoteNone {
			return false
		}
	}
	return true
}

// expandParts processes each span of a word and returns tagged segments.
func expandParts(parts []WordPart) []segment {
	var segs []segment
	for i, p := range parts {
		switch p.Quote {
		case QuoteSingle:
			// Fully literal.
			segs = append(segs, segment{text: p.Text, quoted: true})
		case QuoteDouble:
			// Parameter/command/arithmetic expansion, but no splitting or globbing.
			segs = append(segs, segment{text: expandString(p.Text), quoted: true})
		default: // QuoteNone
			text := p.Text
			// Tilde only at the very start of the word.
			if i == 0 {
				text = expandTilde(text)
			}
			segs = append(segs, segment{text: expandString(text), quoted: false})
		}
	}
	return segs
}

// splitIntoWords performs IFS word-splitting on unquoted segments. Quoted
// segments are attached to the surrounding word without splitting.
func splitIntoWords(segs []segment) [][]segment {
	var words [][]segment
	var current []segment
	started := false

	flush := func() {
		if started {
			words = append(words, current)
			current = nil
			started = false
		}
	}

	for _, s := range segs {
		if s.quoted {
			current = append(current, s)
			started = true
			continue
		}
		// Unquoted: split on runs of whitespace.
		fields := splitKeepEmpty(s.text)
		for idx, f := range fields {
			if idx > 0 {
				flush() // whitespace boundary
			}
			if f != "" {
				current = append(current, segment{text: f, quoted: false})
				started = true
			}
		}
	}
	flush()
	return words
}

// splitKeepEmpty splits on IFS whitespace, keeping the boundaries so the caller
// can tell where words break. Leading/trailing whitespace yields empty leading/
// trailing fields which the caller drops.
func splitKeepEmpty(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n'
	})
}

// globWord assembles a word's segments and, if the whole word is unquoted and
// contains glob metacharacters, expands it against the filesystem. A pattern that
// matches nothing is left literal (bash default, nullglob off).
func globWord(word []segment) []string {
	var b strings.Builder
	onlyUnquoted := true
	for _, s := range word {
		b.WriteString(s.text)
		if s.quoted {
			onlyUnquoted = false
		}
	}
	str := b.String()
	if str == "" {
		return nil
	}
	if onlyUnquoted && hasGlobMeta(str) {
		if matches, err := filepath.Glob(str); err == nil && len(matches) > 0 {
			return matches
		}
	}
	return []string{str}
}

func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// expandTilde resolves a leading ~ or ~user to a home directory.
func expandTilde(s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}
	rest := s[1:]
	// ~ or ~/path -> current user's home.
	if rest == "" || strings.HasPrefix(rest, "/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + rest
		}
		return s
	}
	// ~user or ~user/path
	slash := strings.IndexByte(rest, '/')
	name := rest
	tail := ""
	if slash >= 0 {
		name = rest[:slash]
		tail = rest[slash:]
	}
	if u, err := user.Lookup(name); err == nil {
		return u.HomeDir + tail
	}
	return s
}

var (
	// ${name}, ${name:-word}, ${name-word}, ${#name}
	braceParamRe = regexp.MustCompile(`^\{(#?)([A-Za-z_][A-Za-z0-9_]*|[?$#@*0-9])(:?-[^}]*)?\}`)
	// $name
	bareNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)
)

// expandString performs parameter, command, and arithmetic expansion on a string
// (used for unquoted and double-quoted spans; tilde and splitting are handled by
// the caller). It walks the string so expansions can appear anywhere.
func expandString(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '`' {
			// Backtick command substitution: find the closing backtick.
			if end := strings.IndexByte(s[i+1:], '`'); end >= 0 {
				inner := s[i+1 : i+1+end]
				b.WriteString(captureCommandOutput(inner))
				i = i + 1 + end + 1
				continue
			}
		}
		if c == '$' && i+1 < len(s) {
			next := s[i+1]
			// $(( arithmetic )) or $( command )
			if next == '(' {
				if strings.HasPrefix(s[i:], "$((") {
					if inner, n, ok := matchArith(s[i+3:]); ok {
						b.WriteString(evalArithmetic(inner))
						i += 3 + n
						continue
					}
				}
				if inner, n, ok := matchParens(s[i+2:]); ok {
					b.WriteString(captureCommandOutput(inner))
					i += 2 + n
					continue
				}
			}
			// ${...}
			if next == '{' {
				if m := braceParamRe.FindStringSubmatch(s[i+1:]); m != nil {
					b.WriteString(expandParam(m[1], m[2], m[3]))
					i += 1 + len(m[0])
					continue
				}
			}
			// $name
			if name := bareNameRe.FindString(s[i+1:]); name != "" {
				b.WriteString(expandParam("", name, ""))
				i += 1 + len(name)
				continue
			}
			// $? $$ $# $@ $* $0-$9 (single-char specials)
			if isSpecialParam(next) {
				b.WriteString(expandParam("", string(next), ""))
				i += 2
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func isSpecialParam(b byte) bool {
	return b == '?' || b == '$' || b == '#' || b == '@' || b == '*' || (b >= '0' && b <= '9')
}

// expandParam applies ${#name}, ${name:-word}/${name-word}, or a plain lookup.
func expandParam(hash, name, modifier string) string {
	val, set := lookupVar(name)
	if hash == "#" {
		return strconv.Itoa(len(val))
	}
	if modifier != "" {
		// modifier is ":-word" (use default if unset or empty) or "-word"
		// (use default only if unset).
		colon := strings.HasPrefix(modifier, ":")
		word := strings.TrimPrefix(strings.TrimPrefix(modifier, ":"), "-")
		if colon {
			if !set || val == "" {
				return word
			}
		} else if !set {
			return word
		}
	}
	return val
}

// matchParens returns the content inside a balanced (...) starting at s[0]=='('
// is assumed already consumed by caller; here s begins right after "$(".
func matchParens(s string) (inner string, consumed int, ok bool) {
	depth := 1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[:i], i + 1, true
			}
		}
	}
	return "", 0, false
}

// matchArith finds the end of a $(( )) expression. s begins right after the
// opening "$((", so two parentheses are already open (depth starts at 2). The
// inner expression excludes the final "))". Nested parens inside the expression
// are balanced correctly.
func matchArith(s string) (inner string, consumed int, ok bool) {
	depth := 2
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				// i is the second of the closing "))"; exclude both.
				return s[:i-1], i + 1, true
			}
		}
	}
	return "", 0, false
}

// captureCommandOutput runs a command string through the shell pipeline and
// returns its stdout with trailing newlines trimmed. Builtins are not captured
// in Phase 2 (external commands and pipelines are); this is documented and
// revisited when the executor is unified in Phase 6.
func captureCommandOutput(cmdStr string) string {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return ""
	}
	chain, err := ParseString(cmdStr)
	if err != nil {
		return ""
	}
	ExpandChain(chain)

	var out bytes.Buffer
	for _, pl := range chain.Pipelines {
		runPipelineCapture(pl, &out)
	}
	return strings.TrimRight(out.String(), "\n")
}

// runPipelineCapture executes a single pipeline with its final stdout written to
// buf. It builds exec.Cmds directly (external commands only).
func runPipelineCapture(pl PipelineNode, buf *bytes.Buffer) {
	if len(pl.Commands) == 0 {
		return
	}
	var cmds []*exec.Cmd
	var files []*os.File
	for _, node := range pl.Commands {
		cmd, cleanup, err := buildCommandFromNode(node)
		if err != nil {
			closeFiles(files)
			return
		}
		files = append(files, cleanup...)
		cmds = append(cmds, cmd)
	}
	defer closeFiles(files)

	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			return
		}
		cmds[i+1].Stdin = stdout
	}
	last := cmds[len(cmds)-1]
	if last.Stdout == nil {
		last.Stdout = buf
	}

	for _, c := range cmds {
		if err := c.Start(); err != nil {
			return
		}
	}
	for _, c := range cmds {
		_ = c.Wait()
	}
}

// expandRedirTarget expands a redirection filename (tilde + parameter expansion,
// no splitting or globbing — a redirection target is a single word).
func expandRedirTarget(s string) string {
	if s == "" {
		return s
	}
	return expandString(expandTilde(s))
}
