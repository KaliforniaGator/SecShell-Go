package cmdmap

// Alias expansion. Unlike bash (which expands aliases textually before parsing),
// SecShell expands them post-parse on each command's first word, which covers
// the simple aliases people define (alias ll='ls -la'). A visited set prevents
// infinite recursion on self-referential aliases.

// expandAliases rewrites the first word of every command in the chain when it
// names an alias. It runs before the main expansion stage.
func expandAliases(chain *ChainNode) {
	for pi := range chain.Pipelines {
		for ci := range chain.Pipelines[pi].Commands {
			expandAliasNode(&chain.Pipelines[pi].Commands[ci])
		}
	}
}

func expandAliasNode(cmd *CommandNode) {
	seen := map[string]bool{}
	for {
		if cmd.Name == "" || seen[cmd.Name] {
			return
		}
		// A quoted command word suppresses alias expansion (bash behavior).
		if !allUnquoted(cmd.NameWord) && len(cmd.NameWord) > 0 {
			return
		}
		value, ok := ShellState.GetAlias(cmd.Name)
		if !ok {
			return
		}
		seen[cmd.Name] = true

		// Tokenize the replacement text into words (operators in alias bodies are
		// not supported in this phase).
		var words []Token
		for _, tok := range Tokenize(value) {
			if tok.Type == TokenWord {
				words = append(words, tok)
			}
		}
		if len(words) == 0 {
			return
		}

		// First word becomes the command; the rest are prepended to the args.
		cmd.Name = words[0].Value
		cmd.NameWord = words[0].Parts

		newArgs := make([]string, 0, len(words)-1+len(cmd.Args))
		newArgWords := make([][]WordPart, 0, len(words)-1+len(cmd.ArgWords))
		for _, w := range words[1:] {
			newArgs = append(newArgs, w.Value)
			newArgWords = append(newArgWords, w.Parts)
		}
		cmd.Args = append(newArgs, cmd.Args...)
		cmd.ArgWords = append(newArgWords, cmd.ArgWords...)
	}
}
