package help

import (
	"fmt"
	"secshell/colors"
	"secshell/ui/gui"
)

// DisplayFeatures lists the core features of SecShell.
func DisplayFeatures() {
	gui.TitleBox("SecShell Features")

	features := []string{
		"Secure Command Execution (Whitelisting/Blacklisting)",
		"Input Sanitization & Process Isolation",
		"Background Job Management (`jobs`, `&`)",
		"System Service Management (`services`)",
		"Piped (`|`) and Redirection (`>`, `<`) Support",
		"Advanced Command History (Search, Interactive Mode, `!`, `!!`)",
		"Environment Variable Control (`export`, `unset`, `env`)",
		"Pentesting Suite (`portscan`, `hostscan`, `webscan`, `payload`, `session`)",
		"Encoding/Decoding Utilities (`base64`, `hex`, `url`, `binary`)",
		"Hashing Tools (`hash` - MD5, SHA1, SHA256, SHA512)",
		"Binary Analysis (`extract-strings`)",
		"Script Execution (Auto-detection)",
		"Built-in Pager (`more`)",
		"Built-in Text Editor (`edit`)",
		"Command & Path Auto-Completion (Tab)",
		"Auditable Logging (`logs`)",
		"Self-Update (`--update`) & Version Info (`--version`)",
		"Configurable Security Modes (`toggle-security`)",
		"Customizable Prompt",
		"Date & Time Display (`date`, `time`)",
	}

	fmt.Printf("\n%sCore Features:%s\n", colors.BoldWhite, colors.Reset)
	for _, feature := range features {
		fmt.Printf("  %s✓ %s%s\n", colors.BoldGreen, feature, colors.Reset)
	}
	fmt.Println() // Add a newline at the end for spacing
}
