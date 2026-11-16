package cmdmap

import (
	"fmt"
	"strconv"
	"strings"
)

// TokenType represents the type of a token
type TokenType string

const (
	TokenWord            TokenType = "WORD"             // Command name or argument
	TokenPipe            TokenType = "PIPE"             // |
	TokenAnd             TokenType = "AND"              // &&
	TokenOr              TokenType = "OR"               // ||
	TokenRedirectOut     TokenType = "REDIRECT_OUT"     // >  or N>
	TokenRedirectAppend  TokenType = "REDIRECT_APPEND"  // >> or N>>
	TokenRedirectIn      TokenType = "REDIRECT_IN"      // <  or N<
	TokenRedirectHereDoc TokenType = "REDIRECT_HEREDOC" // <<
	TokenHereString      TokenType = "HERE_STRING"      // <<<
	TokenRedirectBoth    TokenType = "REDIRECT_BOTH"    // &>  (stdout+stderr)
	TokenBackground      TokenType = "BACKGROUND"       // &
	TokenSemi            TokenType = "SEMI"             // ;
	TokenDSemi           TokenType = "DSEMI"            // ;; (case)
	TokenLParen          TokenType = "LPAREN"           // (
	TokenRParen          TokenType = "RPAREN"           // )
)

// QuoteType records how a span of a word was quoted in the source. It is what
// lets a later expansion stage treat $VAR, "$VAR" and '$VAR' differently.
type QuoteType int

const (
	QuoteNone   QuoteType = iota // unquoted text (subject to all expansions)
	QuoteSingle                  // '...'  (no expansion at all)
	QuoteDouble                  // "..."  (parameter/command/arith expansion, no globbing/splitting)
)

// WordPart is a contiguous run of characters from a single word that shared one
// quoting context. A word like a"b"'c' lexes into three parts.
type WordPart struct {
	Text  string
	Quote QuoteType
}

// Token represents a lexed token from the command line.
type Token struct {
	Type  TokenType
	Value string // flattened literal text with quotes removed (kept for back-compat)
	// Parts is set for TokenWord tokens and preserves per-span quoting so the
	// expansion stage (Phase 2) can apply the correct bash rules. Concatenating
	// every Part.Text yields Value.
	Parts []WordPart
	// Fd is the explicit file descriptor on a redirection token (e.g. 2 in 2>),
	// or -1 when none was given.
	Fd int
}

// RedirOp identifies a redirection kind.
type RedirOp int

const (
	RedirIn        RedirOp = iota // <
	RedirOut                      // >
	RedirAppend                   // >>
	RedirBoth                     // &>  (stdout and stderr)
	RedirHeredoc                  // <<  (body filled by the multi-line reader, Phase 5)
	RedirHereStr                  // <<< (Target is the literal string)
)

// Redirection is one redirection on a command. Fd is the affected descriptor
// (0=stdin, 1=stdout, 2=stderr); it defaults from Op when unspecified.
type Redirection struct {
	Fd     int
	Op     RedirOp
	Target string // filename, or the here-string / here-doc delimiter
	Body   string // here-doc body (filled in Phase 5)
}

// CommandNode represents a single command with its arguments and redirections.
type CommandNode struct {
	Name         string   // Command name (flattened)
	Args         []string // Arguments (excluding redirections, flattened)
	StdinFile    string   // File for < input redirection
	StdoutFile   string   // File for > or >> output redirection
	StdoutAppend bool     // true for >> (append), false for > (overwrite)

	// NameWord/ArgWords carry the quote-aware form of Name/Args so a later
	// expansion stage can apply bash expansion rules. They run parallel to the
	// flattened fields: ArgWords[i] corresponds to Args[i]. Executors that do not
	// need quoting information can keep using Name/Args unchanged.
	NameWord []WordPart
	ArgWords [][]WordPart

	// Redirs is the full ordered redirection list, including stderr (2>), combined
	// (&>), and here-strings (<<<) that the legacy StdinFile/StdoutFile fields
	// cannot express. The simple < > >> redirections populate both the legacy
	// fields (for back-compat) and this list.
	Redirs []Redirection
}

// PipelineNode represents a series of commands connected by pipes
type PipelineNode struct {
	Commands   []CommandNode
	Background bool // true if the pipeline ends with &
}

// ChainOperator represents the operator between pipelines
type ChainOperator string

const (
	OpNone ChainOperator = ""   // No operator (single pipeline)
	OpAnd  ChainOperator = "&&" // Run next if previous succeeded
	OpOr   ChainOperator = "||" // Run next if previous failed
	OpSeq  ChainOperator = ";"  // Run next unconditionally (sequence)
)

// ChainNode represents a complete command line with && and || chaining
type ChainNode struct {
	Pipelines []PipelineNode
	Operators []ChainOperator // Operators between pipelines (length = len(Pipelines) - 1)
}

// Tokenize splits a command line into tokens, respecting quotes and escape
// characters. WORD tokens carry per-span quoting information in Token.Parts so a
// later expansion stage can tell $VAR, "$VAR" and '$VAR' apart; Token.Value
// remains the flattened, quote-stripped text.
func Tokenize(input string) []Token {
	var tokens []Token

	// Per-word accumulation. parts holds completed spans; partBuf/partQuote hold
	// the span currently being built. A span is flushed whenever the quoting
	// context changes so each WordPart has a single QuoteType.
	var parts []WordPart
	var partBuf strings.Builder
	var partQuote QuoteType

	flushPart := func() {
		if partBuf.Len() > 0 {
			parts = append(parts, WordPart{Text: partBuf.String(), Quote: partQuote})
			partBuf.Reset()
		}
	}
	// writeChar appends a rune under quote context q, starting a new span when the
	// context differs from the one in progress.
	writeChar := func(r rune, q QuoteType) {
		if partBuf.Len() > 0 && partQuote != q {
			flushPart()
		}
		partQuote = q
		partBuf.WriteRune(r)
	}
	// writeString appends a run of text under one quote context.
	writeString := func(s string, q QuoteType) {
		for _, r := range s {
			writeChar(r, q)
		}
	}
	wordLen := func() int { return partBuf.Len() + len(parts) }
	// takePendingFd recognizes a bare run of digits immediately before a < or >
	// (e.g. the 2 in 2>) as an explicit file descriptor. On a match it consumes
	// the digits (so no word is emitted) and returns the fd; otherwise -1.
	takePendingFd := func() int {
		if len(parts) != 0 || partBuf.Len() == 0 {
			return -1
		}
		s := partBuf.String()
		for _, r := range s {
			if r < '0' || r > '9' {
				return -1
			}
		}
		fd, err := strconv.Atoi(s)
		if err != nil {
			return -1
		}
		partBuf.Reset()
		return fd
	}
	flushWord := func() {
		flushPart()
		if len(parts) > 0 {
			var v strings.Builder
			for _, p := range parts {
				v.WriteString(p.Text)
			}
			tokens = append(tokens, Token{Type: TokenWord, Value: v.String(), Parts: parts})
			parts = nil
		}
	}

	inQuotes := false
	quoteChar := rune(0)
	i := 0
	runes := []rune(input)

	for i < len(runes) {
		char := runes[i]

		// Handle escaped character (highest priority). An escaped character is
		// literal and must not be expanded later, so it is recorded as QuoteSingle
		// regardless of surrounding context.
		if char == '\\' && i+1 < len(runes) {
			writeChar(runes[i+1], QuoteSingle)
			i += 2
			continue
		}

		// Handle quotes
		if char == '"' || char == '\'' {
			if inQuotes && char == quoteChar {
				// End of quoted string
				flushPart()
				inQuotes = false
				quoteChar = rune(0)
				i++
			} else if !inQuotes {
				// Start of quoted string
				flushPart()
				inQuotes = true
				quoteChar = char
				i++
			} else {
				// A different quote character inside quotes is literal.
				q := QuoteSingle
				if quoteChar == '"' {
					q = QuoteDouble
				}
				writeChar(char, q)
				i++
			}
			continue
		}

		// A newline outside quotes is a command separator (treated like ;).
		if char == '\n' && !inQuotes {
			flushWord()
			tokens = append(tokens, Token{Type: TokenSemi, Value: ";"})
			i++
			continue
		}

		// Handle whitespace
		if (char == ' ' || char == '\t') && !inQuotes {
			// Space outside quotes - end current token
			if wordLen() > 0 {
				flushWord()
			}
			i++
			continue
		}

		// Handle whitespace inside quotes
		if (char == ' ' || char == '\t') && inQuotes {
			q := QuoteSingle
			if quoteChar == '"' {
				q = QuoteDouble
			}
			writeChar(char, q)
			i++
			continue
		}

		// Keep substitution spans intact so their inner whitespace is not treated
		// as a word boundary: `...`, $(...), $((...)), ${...}. The whole span is
		// recorded as unquoted text; the expansion stage parses it later.
		if !inQuotes {
			if char == '`' {
				if j := indexRune(runes, i+1, '`'); j >= 0 {
					writeString(string(runes[i:j+1]), QuoteNone)
					i = j + 1
					continue
				}
			}
			if char == '$' && i+1 < len(runes) && (runes[i+1] == '(' || runes[i+1] == '{') {
				if n := scanDollarSpan(runes, i); n > 0 {
					writeString(string(runes[i:i+n]), QuoteNone)
					i += n
					continue
				}
			}
		}

		// Handle operators (only when not in quotes)
		if !inQuotes {
			switch char {
			case '|':
				flushWord()
				if i+1 < len(runes) && runes[i+1] == '|' {
					tokens = append(tokens, Token{Type: TokenOr, Value: "||"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenPipe, Value: "|"})
					i++
				}
				continue

			case '&':
				flushWord()
				if i+1 < len(runes) && runes[i+1] == '&' {
					tokens = append(tokens, Token{Type: TokenAnd, Value: "&&"})
					i += 2
				} else if i+1 < len(runes) && runes[i+1] == '>' {
					tokens = append(tokens, Token{Type: TokenRedirectBoth, Value: "&>", Fd: -1})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenBackground, Value: "&"})
					i++
				}
				continue

			case ';':
				flushWord()
				if i+1 < len(runes) && runes[i+1] == ';' {
					tokens = append(tokens, Token{Type: TokenDSemi, Value: ";;"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenSemi, Value: ";"})
					i++
				}
				continue

			case '(':
				flushWord()
				tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
				i++
				continue

			case ')':
				flushWord()
				tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
				i++
				continue

			case '>':
				fd := takePendingFd()
				flushWord()
				if i+1 < len(runes) && runes[i+1] == '>' {
					tokens = append(tokens, Token{Type: TokenRedirectAppend, Value: ">>", Fd: fd})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenRedirectOut, Value: ">", Fd: fd})
					i++
				}
				continue

			case '<':
				fd := takePendingFd()
				flushWord()
				if strings.HasPrefix(string(runes[i:]), "<<<") {
					tokens = append(tokens, Token{Type: TokenHereString, Value: "<<<", Fd: fd})
					i += 3
				} else if i+1 < len(runes) && runes[i+1] == '<' {
					tokens = append(tokens, Token{Type: TokenRedirectHereDoc, Value: "<<", Fd: fd})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenRedirectIn, Value: "<", Fd: fd})
					i++
				}
				continue
			}
		}

		// Regular character. Inside quotes, tag it with the active quote context so
		// the expansion stage knows whether expansion applies.
		q := QuoteNone
		if inQuotes {
			if quoteChar == '"' {
				q = QuoteDouble
			} else {
				q = QuoteSingle
			}
		}
		writeChar(char, q)
		i++
	}

	// Add the last token if any
	flushWord()

	return tokens
}

// indexRune returns the index of the first occurrence of r in runes at or after
// start, or -1 if not found.
func indexRune(runes []rune, start int, r rune) int {
	for j := start; j < len(runes); j++ {
		if runes[j] == r {
			return j
		}
	}
	return -1
}

// scanDollarSpan measures a $(...), $((...)) or ${...} span beginning at runes[i]
// (which must be '$'). It returns the span length in runes, or 0 if the span is
// not balanced. Parenthesis/brace nesting is respected so $((a+(b*c))) and
// $(echo $(date)) are captured whole.
func scanDollarSpan(runes []rune, i int) int {
	if i+1 >= len(runes) {
		return 0
	}
	var open, close rune
	switch runes[i+1] {
	case '(':
		open, close = '(', ')'
	case '{':
		open, close = '{', '}'
	default:
		return 0
	}
	depth := 0
	for j := i + 1; j < len(runes); j++ {
		switch runes[j] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return j - i + 1
			}
		}
	}
	return 0 // unbalanced
}

// Parse converts a list of tokens into a ChainNode AST
func Parse(tokens []Token) (*ChainNode, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	chain := &ChainNode{}
	var currentPipeline PipelineNode
	var currentCmd CommandNode
	var pendingRedir *Redirection

	for _, token := range tokens {
		if pendingRedir != nil {
			// This token is the target (filename or here-string) for the redirection.
			pendingRedir.Target = token.Value
			// Mirror simple descriptor-default redirections into the legacy fields
			// the current executor reads; fd-specific ones (e.g. 2>) live only in
			// the Redirs list.
			switch pendingRedir.Op {
			case RedirIn:
				if pendingRedir.Fd == -1 || pendingRedir.Fd == 0 {
					currentCmd.StdinFile = token.Value
				}
			case RedirOut:
				if pendingRedir.Fd == -1 || pendingRedir.Fd == 1 {
					currentCmd.StdoutFile = token.Value
					currentCmd.StdoutAppend = false
				}
			case RedirAppend:
				if pendingRedir.Fd == -1 || pendingRedir.Fd == 1 {
					currentCmd.StdoutFile = token.Value
					currentCmd.StdoutAppend = true
				}
			}
			currentCmd.Redirs = append(currentCmd.Redirs, *pendingRedir)
			pendingRedir = nil
			continue
		}

		switch token.Type {
		case TokenWord:
			if currentCmd.Name == "" && currentCmd.NameWord == nil {
				currentCmd.Name = token.Value
				currentCmd.NameWord = token.Parts
			} else {
				currentCmd.Args = append(currentCmd.Args, token.Value)
				currentCmd.ArgWords = append(currentCmd.ArgWords, token.Parts)
			}

		case TokenPipe:
			// End current command and start a new one in this pipeline
			if currentCmd.Name != "" {
				currentPipeline.Commands = append(currentPipeline.Commands, currentCmd)
				currentCmd = CommandNode{}
			}

		case TokenAnd, TokenOr, TokenSemi:
			// End current pipeline and start a new one
			if currentCmd.Name != "" {
				currentPipeline.Commands = append(currentPipeline.Commands, currentCmd)
				currentCmd = CommandNode{}
			}
			if len(currentPipeline.Commands) > 0 {
				chain.Pipelines = append(chain.Pipelines, currentPipeline)
				currentPipeline = PipelineNode{}
			}
			switch token.Type {
			case TokenAnd:
				chain.Operators = append(chain.Operators, OpAnd)
			case TokenOr:
				chain.Operators = append(chain.Operators, OpOr)
			default:
				chain.Operators = append(chain.Operators, OpSeq)
			}

		case TokenBackground:
			// Mark this pipeline as background
			currentPipeline.Background = true

		case TokenRedirectIn:
			pendingRedir = &Redirection{Op: RedirIn, Fd: token.Fd}
		case TokenRedirectOut:
			pendingRedir = &Redirection{Op: RedirOut, Fd: token.Fd}
		case TokenRedirectAppend:
			pendingRedir = &Redirection{Op: RedirAppend, Fd: token.Fd}
		case TokenRedirectBoth:
			pendingRedir = &Redirection{Op: RedirBoth, Fd: -1}
		case TokenHereString:
			pendingRedir = &Redirection{Op: RedirHereStr, Fd: token.Fd}

		case TokenRedirectHereDoc:
			// Here-docs need the multi-line reader (Phase 5); the compound-command
			// parser (ParseProgram) records them for that path.
			return nil, fmt.Errorf("here-doc (<<) requires multi-line input")

		default:
			// LParen/RParen/DSemi and other compound tokens are handled by the
			// recursive-descent parser (ParseProgram), not this simple path.
			return nil, fmt.Errorf("unexpected token type: %s", token.Type)
		}
	}

	// Add the last command to the pipeline
	if currentCmd.Name != "" || len(currentPipeline.Commands) > 0 {
		if currentCmd.Name != "" {
			currentPipeline.Commands = append(currentPipeline.Commands, currentCmd)
		}
		chain.Pipelines = append(chain.Pipelines, currentPipeline)
	}

	if len(chain.Pipelines) == 0 {
		return nil, fmt.Errorf("no commands found in input")
	}

	// Drop any dangling operators left by a trailing ; && || (e.g. "ls ;") so
	// len(Operators) stays len(Pipelines)-1, which executeChain relies on.
	for len(chain.Operators) >= len(chain.Pipelines) && len(chain.Operators) > 0 {
		chain.Operators = chain.Operators[:len(chain.Operators)-1]
	}

	return chain, nil
}

// ParseString is a convenience function that tokenizes and parses a command string
func ParseString(input string) (*ChainNode, error) {
	tokens := Tokenize(input)
	return Parse(tokens)
}