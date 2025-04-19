![Preview](https://github.com/user-attachments/assets/c5444355-b435-4fbf-84a0-cf1f1ad23233)

# 🚨 SecShell - Secure Shell for Modern Systems (Go)

**SecShell** is a next-generation secure shell written in Go, engineered for professionals who demand robust security, fine-grained control, and operational transparency. It provides a hardened environment for command execution, featuring advanced whitelisting, process isolation, and real-time job/service management.

---

## 🔑 Key Features

- **Command Whitelisting & Blacklisting**: Only explicitly allowed commands or those in trusted directories can run. Blacklisted commands are strictly blocked.
- **Input Sanitization**: All user input is sanitized to prevent injection and exploitation.
- **Process Isolation**: Each command runs in its own process, minimizing risk.
- **Job Management**: Track, control, and inspect background jobs.
- **Service Management**: Start, stop, restart, and check system services securely.
- **Piped & Background Execution**: Full support for pipes (`|`), redirection (`>`, `<`), and background jobs (`&`).
- **Command History & Search**: Persistent history with interactive and query-based search.
- **Environment Variable Control**: Set, unset, and list environment variables.
- **Security Toggle (Admin Only)**: Temporarily bypass security checks with authentication.
- **Pentesting Utilities**: Built-in port, host, and web scanners, plus reverse shell payload generation and session management.
- **Update & Version Control**: Self-update and version display commands.
- **Comprehensive Logging**: All actions are logged for audit and review.

---

## 🛡️ Built-in Commands
________________________________________________________________________________________________________________________________
| Command                | Description / Usage                                                                                 |
|------------------------|-----------------------------------------------------------------------------------------------------|
| `allowed`              | Show allowed directories, commands, built-ins, or binaries.<br>                                     |
|                        | Usage: `allowed <dirs|commands|bins|builtins|all>`                                                  |
| `help`                 | Show help message or help for a specific command.<br>Usage: `help [command]`                        |
| `exit`                 | Exit the shell (admin only).                                                                        |
| `services`             | Manage system services.<br>Usage: `services <start|stop|restart|status|list> <service_name>`        |
| `jobs`                 | Manage background jobs.<br>Usage: `jobs <list|stop|start|status|clear-finished> [pid]`              |
| `cd`                   | Change directory.<br>Usage: `cd [directory]`                                                        |
| `history`              | Show command history.<br>Usage: `history`, `history -s <query>`, `history -i`, `history clear`      |
| `export`               | Set an environment variable.<br>Usage: `export VAR=value`                                           |
| `env`                  | List all environment variables.                                                                     |
| `unset`                | Unset an environment variable.<br>Usage: `unset VAR`                                                |
| `logs`                 | List or clear logs.<br>Usage: `logs <list|clear>`                                                   |
| `blacklist`            | List blacklisted commands.                                                                          |
| `whitelist`            | List whitelisted commands.                                                                          |
| `edit-blacklist`       | Edit the blacklist file (admin only).                                                               |
| `edit-whitelist`       | Edit the whitelist file (admin only).                                                               |
| `reload-blacklist`     | Reload the blacklist (admin only).                                                                  |
| `reload-whitelist`     | Reload the whitelist (admin only).                                                                  |
| `download`             | Download files from the internet.<br>Usage: `download [-o output1,output2,...] <url [url2 ...]>`    |
| `toggle-security`      | Toggle security enforcement (admin only, password required).                                        |
| `time`                 | Show current time.                                                                                  |
| `date`                 | Show current date.                                                                                  |
| `--version`            | Display current version.                                                                            |
| `--update`             | Update SecShell to the latest version.                                                              |
| **Pentesting Tools**   |                                                                                                     |
| `portscan`             | Advanced port scanner.<br>Usage: `portscan [options] <target>`<br>                                 |
|                        | Options:<br>                                                                                        |
|                        | &nbsp;&nbsp;`-p <ports>` (port range, e.g. 1-1000)<br>                                              |
|                        | &nbsp;&nbsp;`-udp` (UDP scan)<br>                                                                   |
|                        | &nbsp;&nbsp;`-t <1-5>` (timing, 1=slowest, 5=fastest)<br>                                           |
|                        | &nbsp;&nbsp;`-v` (show service version)<br>                                                         |
|                        | &nbsp;&nbsp;`-j` (JSON output), `-html` (HTML output)<br>                                           |
|                        | &nbsp;&nbsp;`-o <file>` (output file)<br>                                                           |
|                        | &nbsp;&nbsp;`-syn` (SYN scan, root only)<br>                                                        |
|                        | &nbsp;&nbsp;`-os` (OS detection)<br>                                                                |
|                        | &nbsp;&nbsp;`-e` (enhanced detection)<br>                                                           |
| `hostscan`             | Discover hosts in a network.<br>Usage: `hostscan <network-range>`                                   |
| `webscan`              | Scan a web target.<br>Usage: `webscan [options] <url>`<br>                                          |
|                        | Options:<br>                                                                                        |
|                        | &nbsp;&nbsp;`-t, --timeout <sec>`<br>                                                               |
|                        | &nbsp;&nbsp;`-H, --header <Header:Value>`<br>                                                       |
|                        | &nbsp;&nbsp;`-k, --insecure` (skip SSL verification)<br>                                            |
|                        | &nbsp;&nbsp;`-A, --user-agent <UA>`<br>                                                             |
|                        | &nbsp;&nbsp;`--threads <n>`<br>                                                                     |
|                        | &nbsp;&nbsp;`-w, --wordlist <file>`<br>                                                             |
|                        | &nbsp;&nbsp;`-m, --methods <GET,POST,...>`<br>                                                      |
|                        | &nbsp;&nbsp;`-v, --verbose`<br>                                                                     |
|                        | &nbsp;&nbsp;`--follow-redirects`<br>                                                                |
|                        | &nbsp;&nbsp;`--cookie <cookie>`<br>                                                                 |
|                        | &nbsp;&nbsp;`--auth <user:pass>`<br>                                                                |
|                        | &nbsp;&nbsp;`-f, --format <text|json|html>`<br>                                                     |
|                        | &nbsp;&nbsp;`-o, --output <file>`<br>                                                               |
| `payload`              | Generate reverse shell payload.<br>Usage: `payload <ip-address> <port>`                             |
| `session`              | Manage pentest sessions.<br>                                                                        |
|                        | Usage:<br>                                                                                          |
|                        | &nbsp;&nbsp;`session -l` (list sessions)<br>                                                        |
|                        | &nbsp;&nbsp;`session -i <id>` (interact with session)<br>                                           |
|                        | &nbsp;&nbsp;`session -c <port>` (listen for new session)<br>                                        |
|                        | &nbsp;&nbsp;`session -k <id>` (kill session)                                                        |
--------------------------------------------------------------------------------------------------------------------------------
---

## ⚡ Quick Start

### Requirements

- **Go (Golang)**
- **systemctl**
- **Nano Editor**
- **DrawBox** ([DrawBox Repository](https://github.com/KaliforniaGator/DrawBox))
- **PAM Development Library (`libpam0g-dev`)**

### One-Step Installation

```bash
curl -fsSL https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/main/update.sh | bash
```

This will install dependencies, clone SecShell-Go, and build the project.

---

## 🚀 Usage

Start SecShell:

```bash
secshell
```

### Example Commands

- List files: `ls -l`
- Start a service: `services start nginx`
- Set an environment variable: `export MY_VAR=value`
- Run a command in the background: `sleep 10 &`
- View command history: `history`
- Search history: `history -s nginx`
- Interactive history search: `history -i`
- Download a file: `download https://example.com/file.txt`
- Scan ports: `portscan 192.168.1.1 1-1000`
- Toggle security (admin): `toggle-security`

---

## ⚙️ Configuration

SecShell uses two config files:

- `.whitelist` — List of allowed commands.
- `.blacklist` — List of disallowed commands.

Edit with `edit-whitelist` or `edit-blacklist` (admin only). Files are auto-created if missing.

---

## 🔒 Security Model

- **Strict Whitelisting**: Only commands in `.whitelist` or trusted directories are allowed.
- **Blacklist Enforcement**: Blacklisted commands are always blocked.
- **Admin Bypass**: Admins can temporarily disable security (with authentication).
- **Network Command Restrictions**: Sensitive network tools (e.g., `wget`, `curl`, `nmap`) are restricted for non-admins.
- **Audit Logging**: All actions are logged for review.

---

## 🤝 Contributing

Contributions are welcome! Please open issues or submit pull requests for improvements, bug fixes, or new features.

---

## 📄 License

SecShell is licensed under the **GNU Affero General Public License (AGPL)**. See [LICENSE](LICENSE) for details.

---

**Serious about security. Built for professionals.**
