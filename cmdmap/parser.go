package cmdmap

import (
	"fmt"
	"strings"
)

// TokenType represents the type of a token
type TokenType string

const (
	TokenWord       TokenType = "WORD"       // Command name or argument
	TokenPipe       TokenType = "PIPE"       // |
	TokenAnd        TokenType = "AND"        // &&
	TokenOr         TokenType = "OR"         // ||
	TokenRedirectOut TokenType = "REDIRECT_OUT"  // >
	TokenRedirectAppend TokenType = "REDIRECT_APPEND" // >>
	TokenRedirectIn TokenType = "REDIRECT_IN"   // <
	TokenRedirectHereDoc TokenType = "REDIRECT_HEREDOC" // <<
	TokenBackground TokenType = "BACKGROUND" // &
)

// Token represents a lexed token from the command line
type Token struct {
	Type  TokenType
	Value string
}

// CommandNode represents a single command with its arguments and redirections
type CommandNode struct {
	Name         string   // Command name
	Args         []string // Arguments (excluding redirections)
	StdinFile    string   // File for < input redirection
	StdoutFile   string   // File for > or >> output redirection
	StdoutAppend bool     // true for >> (append), false for > (overwrite)
}

// PipelineNode represents a series of commands connected by pipes
type PipelineNode struct {
	Commands   []CommandNode
	Background bool // true if the pipeline ends with &
}

// ChainOperator represents the operator between pipelines
type ChainOperator string

const (
	OpNone ChainOperator = ""     // No operator (single pipeline)
	OpAnd  ChainOperator = "&&"  // Run next if previous succeeded
	OpOr   ChainOperator = "||"  // Run next if previous failed
)

// ChainNode represents a complete command line with && and || chaining
type ChainNode struct {
	Pipelines []PipelineNode
	Operators []ChainOperator // Operators between pipelines (length = len(Pipelines) - 1)
}

// Tokenize splits a command line into tokens, respecting quotes and escape characters
func Tokenize(input string) []Token {
	var tokens []Token
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	i := 0
	runes := []rune(input)

	for i < len(runes) {
		char := runes[i]

		// Handle escaped character (highest priority)
		if char == '\\' && i+1 < len(runes) {
			// Write the next character literally (backslash escape)
			current.WriteRune(runes[i+1])
			i += 2
			continue
		}

		// Handle quotes
		if char == '"' || char == '\'' {
			if inQuotes && char == quoteChar {
				// End of quoted string
				inQuotes = false
				quoteChar = rune(0)
				i++
			} else if !inQuotes {
				// Start of quoted string
				inQuotes = true
				quoteChar = char
				i++
			} else {
				// Different quote inside quotes
				current.WriteRune(char)
				i++
			}
			continue
		}

		// Handle whitespace
		if (char == ' ' || char == '\t') && !inQuotes {
			// Space outside quotes - end current token
			if current.Len() > 0 {
				tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
				current.Reset()
			}
			i++
			continue
		}

		// Handle whitespace inside quotes
		if (char == ' ' || char == '\t') && inQuotes {
			current.WriteRune(char)
			i++
			continue
		}

		// Handle operators (only when not in quotes)
		if !inQuotes {
			switch char {
			case '|':
				if current.Len() > 0 {
					tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
					current.Reset()
				}
				if i+1 < len(runes) && runes[i+1] == '|' {
					tokens = append(tokens, Token{Type: TokenOr, Value: "||"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenPipe, Value: "|"})
					i++
				}
				continue

			case '&':
				if current.Len() > 0 {
					tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
					current.Reset()
				}
				if i+1 < len(runes) && runes[i+1] == '&' {
					tokens = append(tokens, Token{Type: TokenAnd, Value: "&&"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenBackground, Value: "&"})
					i++
				}
				continue

			case '>':
				if current.Len() > 0 {
					tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
					current.Reset()
				}
				if i+1 < len(runes) && runes[i+1] == '>' {
					tokens = append(tokens, Token{Type: TokenRedirectAppend, Value: ">>"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenRedirectOut, Value: ">"})
					i++
				}
				continue

			case '<':
				if current.Len() > 0 {
					tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
					current.Reset()
				}
				if i+1 < len(runes) && runes[i+1] == '<' {
					tokens = append(tokens, Token{Type: TokenRedirectHereDoc, Value: "<<"})
					i += 2
				} else {
					tokens = append(tokens, Token{Type: TokenRedirectIn, Value: "<"})
					i++
				}
				continue
			}
		}

		// Regular character (or character inside quotes)
		current.WriteRune(char)
		i++
	}

	// Add the last token if any
	if current.Len() > 0 {
		tokens = append(tokens, Token{Type: TokenWord, Value: current.String()})
	}

	return tokens
}

// Parse converts a list of tokens into a ChainNode AST
func Parse(tokens []Token) (*ChainNode, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	chain := &ChainNode{}
	var currentPipeline PipelineNode
	var currentCmd CommandNode
	expectingFilename := false
	redirectionType := ""

	for _, token := range tokens {
		if expectingFilename {
			// This token is the filename for the redirection
			switch redirectionType {
			case "<":
				currentCmd.StdinFile = token.Value
			case ">":
				currentCmd.StdoutFile = token.Value
				currentCmd.StdoutAppend = false
			case ">>":
				currentCmd.StdoutFile = token.Value
				currentCmd.StdoutAppend = true
			}
			expectingFilename = false
			redirectionType = ""
			continue
		}

		switch token.Type {
		case TokenWord:
			if currentCmd.Name == "" {
				currentCmd.Name = token.Value
			} else {
				currentCmd.Args = append(currentCmd.Args, token.Value)
			}

		case TokenPipe:
			// End current command and start a new one in this pipeline
			if currentCmd.Name != "" {
				currentPipeline.Commands = append(currentPipeline.Commands, currentCmd)
				currentCmd = CommandNode{}
			}

		case TokenAnd, TokenOr:
			// End current pipeline and start a new one
			if currentCmd.Name != "" {
				currentPipeline.Commands = append(currentPipeline.Commands, currentCmd)
				currentCmd = CommandNode{}
			}
			if len(currentPipeline.Commands) > 0 {
				chain.Pipelines = append(chain.Pipelines, currentPipeline)
				currentPipeline = PipelineNode{}
			}
			if token.Type == TokenAnd {
				chain.Operators = append(chain.Operators, OpAnd)
			} else {
				chain.Operators = append(chain.Operators, OpOr)
			}

		case TokenBackground:
			// Mark this pipeline as background
			currentPipeline.Background = true

		case TokenRedirectIn, TokenRedirectOut, TokenRedirectAppend:
			// Next token should be a filename
			expectingFilename = true
			if token.Type == TokenRedirectIn {
				redirectionType = "<"
			} else if token.Type == TokenRedirectOut {
				redirectionType = ">"
			} else {
				redirectionType = ">>"
			}

		case TokenRedirectHereDoc:
			// For now, here-doc support is limited - we'll handle it as an error
			// A full implementation would read until the delimiter
			return nil, fmt.Errorf("here-doc (<<) is not yet fully supported")

		default:
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

	return chain, nil
}

// ParseString is a convenience function that tokenizes and parses a command string
func ParseString(input string) (*ChainNode, error) {
	tokens := Tokenize(input)
	return Parse(tokens)
}