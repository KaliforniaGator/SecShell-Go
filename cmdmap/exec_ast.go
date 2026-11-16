package cmdmap

import (
	"os"
	"path/filepath"
	"strconv"

	"secshell/jobs"
	"secshell/ui/gui"
)

// Phase 5: the AST executor. It walks the program AST from ParseProgram and
// handles control flow (if/for/while/until/case, functions, break/continue/
// return). Leaf simple commands and multi-command pipelines are delegated to the
// existing execution machinery (executeSingleCommand / executePipeline / the
// background and more-pager paths), so redirection, builtins, job control and
// signal handling are reused rather than reimplemented.

type flowKind int

const (
	flowNone flowKind = iota
	flowBreak
	flowContinue
	flowReturn
)

type astExec struct {
	jobsMap map[int]*jobs.Job
	last    int      // exit status of the last command ($?)
	flow    flowKind // pending non-local control transfer
	flowN   int      // remaining loop levels for break/continue
}

// ExecuteProgram runs a parsed program and returns the final exit status.
func ExecuteProgram(list *ListNode, jobsMap map[int]*jobs.Job) int {
	e := &astExec{jobsMap: jobsMap}
	e.execList(list)
	lastExitStatus = e.last
	return e.last
}

func (e *astExec) execList(list *ListNode) {
	for _, item := range list.Items {
		if e.flow != flowNone {
			return
		}
		e.execAndOr(item.AndOr, item.Background)
		lastExitStatus = e.last
	}
}

func (e *astExec) execAndOr(ao *AndOrNode, bg bool) {
	for i, pipe := range ao.Pipelines {
		if i > 0 {
			switch ao.Ops[i-1] {
			case OpAnd:
				if e.last != 0 {
					return
				}
			case OpOr:
				if e.last == 0 {
					return
				}
			}
		}
		e.execPipeline(pipe, bg && i == len(ao.Pipelines)-1)
		if e.flow != flowNone {
			return
		}
	}
}

func (e *astExec) execPipeline(pipe *PipeNode, bg bool) {
	if len(pipe.Cmds) == 1 {
		e.execCommandNode(pipe.Cmds[0], bg)
		return
	}
	// Multi-command pipeline: only simple commands are supported as pipe stages.
	var cmds []CommandNode
	for _, c := range pipe.Cmds {
		sc, ok := c.(*SimpleCmd)
		if !ok {
			gui.ErrorBox("pipelines containing compound commands are not supported yet")
			e.last = 1
			return
		}
		cn := simpleToCommandNode(sc)
		expandAliasNode(&cn)
		expandCommandNode(&cn)
		cmds = append(cmds, cn)
	}
	pl := PipelineNode{Commands: cmds, Background: bg}
	switch {
	case bg:
		executePipelineBackground(pl, e.jobsMap, &e.last)
	case cmds[len(cmds)-1].Name == "more":
		executePipelineWithMore(cmds[:len(cmds)-1], &e.last)
	default:
		executePipeline(pl, &e.last)
	}
}

func (e *astExec) execCommandNode(node Node, bg bool) {
	switch n := node.(type) {
	case *SimpleCmd:
		e.execSimple(n, bg)
	case *IfNode:
		e.execIf(n)
	case *ForNode:
		e.execFor(n)
	case *WhileNode:
		e.execWhile(n)
	case *CaseNode:
		e.execCase(n)
	case *FuncNode:
		ShellState.SetFunc(n.Name, n.Body)
		e.last = 0
	case *SubshellNode:
		e.execSubshell(n)
	case *GroupNode:
		e.execList(n.Body)
	default:
		gui.ErrorBox("unsupported command type")
		e.last = 1
	}
}

func (e *astExec) execSimple(sc *SimpleCmd, bg bool) {
	// Loop/function control keywords are intercepted before expansion so they can
	// drive the executor's flow state.
	switch sc.Name {
	case "break":
		e.flow = flowBreak
		e.flowN = intArg(sc, 1)
		e.last = 0
		return
	case "continue":
		e.flow = flowContinue
		e.flowN = intArg(sc, 1)
		e.last = 0
		return
	case "return":
		e.flow = flowReturn
		if len(sc.Args) > 0 {
			if n, err := strconv.Atoi(sc.Args[0]); err == nil {
				e.last = n
			}
		}
		return
	}

	cn := simpleToCommandNode(sc)
	expandAliasNode(&cn)
	expandCommandNode(&cn)

	// A user-defined function takes precedence over external commands.
	if body, ok := ShellState.GetFunc(cn.Name); ok {
		e.callFunction(body, cn.Args)
		return
	}

	if bg {
		executeSingleCommandBackground(cn, e.jobsMap)
		e.last = 0
		return
	}
	e.last = executeSingleCommand(cn)
}

func (e *astExec) callFunction(body Node, args []string) {
	// A function is a flow boundary: it gets fresh break/continue state, its own
	// positional parameters, and return only unwinds the function.
	savedPos := ShellState.Positional()
	savedFlow, savedN := e.flow, e.flowN
	e.flow, e.flowN = flowNone, 0
	ShellState.SetPositional(args)

	e.execCommandNode(body, false)

	ShellState.SetPositional(savedPos)
	e.flow, e.flowN = savedFlow, savedN
}

func (e *astExec) execIf(n *IfNode) {
	e.execList(n.Cond)
	if e.flow != flowNone {
		return
	}
	if e.last == 0 {
		e.execList(n.Then)
		return
	}
	for _, elif := range n.Elifs {
		e.execList(elif.Cond)
		if e.flow != flowNone {
			return
		}
		if e.last == 0 {
			e.execList(elif.Then)
			return
		}
	}
	if n.Else != nil {
		e.execList(n.Else)
	}
}

func (e *astExec) execFor(n *ForNode) {
	var words []string
	for _, w := range n.Words {
		words = append(words, expandWord(w.Parts, w.Value)...)
	}
	for _, w := range words {
		ShellState.SetVar(n.Var, w)
		e.execList(n.Body)
		if stop := e.loopFlow(); stop {
			return
		}
	}
}

func (e *astExec) execWhile(n *WhileNode) {
	for {
		e.execList(n.Cond)
		if e.flow != flowNone {
			return
		}
		stop := (!n.Until && e.last != 0) || (n.Until && e.last == 0)
		if stop {
			return
		}
		e.execList(n.Body)
		if s := e.loopFlow(); s {
			return
		}
	}
}

// loopFlow resolves break/continue at the end of a loop iteration. It returns
// true when the loop should stop (break, or a continue/break targeting an outer
// loop, or a return/other flow that must propagate).
func (e *astExec) loopFlow() bool {
	switch e.flow {
	case flowBreak:
		e.flowN--
		if e.flowN <= 0 {
			e.flow = flowNone
		}
		return true // this loop stops; an outer loop stops too if flow remains
	case flowContinue:
		e.flowN--
		if e.flowN <= 0 {
			e.flow = flowNone
			return false // continue this loop
		}
		return true // continue targets an outer loop: stop this one, propagate
	case flowReturn:
		return true
	}
	return false
}

func (e *astExec) execCase(n *CaseNode) {
	word := expandToString(n.Word.Parts, n.Word.Value)
	for _, item := range n.Items {
		for _, pat := range item.Patterns {
			p := expandToString(pat.Parts, pat.Value)
			if p == "*" {
				e.execList(item.Body)
				return
			}
			if matched, err := filepath.Match(p, word); err == nil && matched {
				e.execList(item.Body)
				return
			}
		}
	}
}

func (e *astExec) execSubshell(n *SubshellNode) {
	// Basic subshell isolation: changes to the working directory and shell
	// variables are reverted when the subshell exits, and control flow does not
	// escape it. (Full process isolation is a later refinement.)
	savedCwd, _ := os.Getwd()
	savedVars := make(map[string]string, len(ShellState.vars))
	for k, v := range ShellState.vars {
		savedVars[k] = v
	}

	e.execList(n.Body)

	if savedCwd != "" {
		_ = os.Chdir(savedCwd)
	}
	ShellState.vars = savedVars
	if e.flow != flowReturn {
		e.flow, e.flowN = flowNone, 0
	}
}

// --- helpers ---

// simpleToCommandNode converts a parsed SimpleCmd into the CommandNode the
// existing executor consumes, mirroring default-descriptor redirections into the
// legacy StdinFile/StdoutFile fields (fd-specific ones stay in Redirs).
func simpleToCommandNode(sc *SimpleCmd) CommandNode {
	cn := CommandNode{
		Name:     sc.Name,
		NameWord: sc.NameWord,
		Args:     sc.Args,
		ArgWords: sc.ArgWords,
		Redirs:   sc.Redirs,
	}
	for _, r := range sc.Redirs {
		switch r.Op {
		case RedirIn:
			if r.Fd == -1 || r.Fd == 0 {
				cn.StdinFile = r.Target
			}
		case RedirOut:
			if r.Fd == -1 || r.Fd == 1 {
				cn.StdoutFile = r.Target
				cn.StdoutAppend = false
			}
		case RedirAppend:
			if r.Fd == -1 || r.Fd == 1 {
				cn.StdoutFile = r.Target
				cn.StdoutAppend = true
			}
		}
	}
	return cn
}

// intArg parses the first argument of a command as an integer, or returns def.
func intArg(sc *SimpleCmd, def int) int {
	if len(sc.Args) > 0 {
		if n, err := strconv.Atoi(sc.Args[0]); err == nil {
			return n
		}
	}
	return def
}
