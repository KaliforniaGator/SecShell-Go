package cmdmap

// Phase 4: the full program AST produced by the recursive-descent parser
// (ParseProgram in grammar.go). It expresses the constructs the simple Parse
// cannot — sequencing, compound commands, and function definitions. It is not
// yet wired into execution; Phase 5 adds the executor and multi-line reader and
// switches ExecuteCommand over to it.

// Node is any program AST node.
type Node interface{ node() }

// WordSpec is one word with its quote-aware parts, for later expansion.
type WordSpec struct {
	Value string
	Parts []WordPart
}

// ListNode is a sequence of and-or lists separated by ; & or newline.
type ListNode struct {
	Items []AndOrItem
}

// AndOrItem is one element of a list, optionally run in the background (&).
type AndOrItem struct {
	AndOr      *AndOrNode
	Background bool
}

// AndOrNode is pipelines joined by && / ||. Ops has len(Pipelines)-1 entries.
type AndOrNode struct {
	Pipelines []*PipeNode
	Ops       []ChainOperator
}

// PipeNode is commands joined by |.
type PipeNode struct {
	Cmds []Node // each *SimpleCmd or a compound node
}

// SimpleCmd is a plain command with arguments and redirections.
type SimpleCmd struct {
	Name     string
	NameWord []WordPart
	Args     []string
	ArgWords [][]WordPart
	Redirs   []Redirection
}

// IfNode: if Cond; then Then; [elif ...]; [else Else]; fi
type IfNode struct {
	Cond   *ListNode
	Then   *ListNode
	Elifs  []ElifPart
	Else   *ListNode
	Redirs []Redirection
}

type ElifPart struct {
	Cond *ListNode
	Then *ListNode
}

// ForNode: for Var in Words; do Body; done
type ForNode struct {
	Var    string
	Words  []WordSpec
	Body   *ListNode
	Redirs []Redirection
}

// WhileNode: while/until Cond; do Body; done
type WhileNode struct {
	Cond   *ListNode
	Body   *ListNode
	Until  bool
	Redirs []Redirection
}

// CaseNode: case Word in (pat|pat) body ;; ... esac
type CaseNode struct {
	Word   WordSpec
	Items  []CaseItem
	Redirs []Redirection
}

type CaseItem struct {
	Patterns []WordSpec
	Body     *ListNode
}

// FuncNode: name() compound  OR  function name compound
type FuncNode struct {
	Name string
	Body Node // a compound command (group, subshell, or other)
}

// SubshellNode: ( list )
type SubshellNode struct {
	Body   *ListNode
	Redirs []Redirection
}

// GroupNode: { list ; }
type GroupNode struct {
	Body   *ListNode
	Redirs []Redirection
}

func (*ListNode) node()     {}
func (*AndOrNode) node()    {}
func (*PipeNode) node()     {}
func (*SimpleCmd) node()    {}
func (*IfNode) node()       {}
func (*ForNode) node()      {}
func (*WhileNode) node()    {}
func (*CaseNode) node()     {}
func (*FuncNode) node()     {}
func (*SubshellNode) node() {}
func (*GroupNode) node()    {}
