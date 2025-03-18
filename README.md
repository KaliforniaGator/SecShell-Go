# SecShell - A Secure Shell Implementation in Go

SecShell is a secure shell implementation written in Go, designed to provide a controlled and secure environment for executing system commands. It incorporates various security features such as command whitelisting, input sanitization, and process isolation to ensure that only trusted commands are executed.

## Features

- **Command Whitelisting**: Only commands listed in the whitelist or located in trusted directories are allowed to execute.
- **Input Sanitization**: Removes potentially harmful characters from user input to prevent injection attacks.
- **Process Isolation**: Executes commands in isolated processes to minimize the risk of system compromise.
- **Job Tracking**: Tracks background jobs and allows users to manage them.
- **Service Management**: Provides commands to start, stop, restart, and check the status of system services.
- **Background Job Execution**: Supports running commands in the background.
- **Piped Command Execution**: Allows chaining commands using pipes.
- **Input/Output Redirection**: Supports input and output redirection for commands.
- **Command History**: Keeps a history of executed commands for easy retrieval.
- **Environment Variable Management**: Allows users to set, unset, and list environment variables.

## Installation

###Requirements
1. GoLang-Go
   ```bash
   sudo apt install golang-go
   ```  
2. systemctl
   ```bash
   sudo apt install systemctl
   ```
3. drawbox
   [Installation Instructions](https://github.com/KaliforniaGator/DrawBox)

4. PAM-Dev
   ```sh
   sudo apt update
   sudo apt install libpam0g-dev

   ```

To use SecShell, you need to have Go installed on your system. Follow these steps to get started:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/secshell.git
   cd secshell
   ```

2. **Build the project**:
   ```bash
   go build -o secshell
   ```

3. **Run the shell**:
   ```bash
   ./secshell
   ```

## Usage

Once the shell is running, you can start executing commands. The shell supports a variety of built-in commands and system commands.

### Built-in Commands

- **help**: Show the help message.
- **exit**: Exit the shell.
- **services**: Manage system services.
  - Usage: `services <start|stop|restart|status|list> <service_name>`
- **jobs**: List active background jobs.
- **cd**: Change directory.
  - Usage: `cd [directory]`
- **history**: Show command history.
  - Usage: To search `history [-s query]` To use the interactive history `history -i`
- **export**: Set an environment variable.
  - Usage: `export VAR=value`
- **env**: List all environment variables.
- **unset**: Unset an environment variable.
  - Usage: `unset VAR`
- **blacklist**: List blacklisted commands.
- **whitelist**: List whitelisted commands.
- **edit-blacklist**: Edit the blacklist file.
- **edit-whitelist**: Edit the whitelist file.
- **reload-blacklist**: Reload the blacklisted commands.
- **reload-whitelist**: Reload the whitelisted commands.
- **download** Download a file from the internet using: download <filename>.
- **toggle-security** Run commands as an administrator bypassing the whitelisting and blacklisting.

### Examples

- List files in the current directory:
  ```bash
  ls -l
  ```

- Start a service:
  ```bash
  services start nginx
  ```

- Set an environment variable:
  ```bash
  export MY_VAR=value
  ```

- Run a command in the background:
  ```bash
  sleep 10 &
  ```

- View command history:
  ```bash
  history
  ```

## Configuration

SecShell uses two configuration files to manage allowed and disallowed commands:

- **.whitelist**: Contains a list of allowed commands.
- **.blacklist**: Contains a list of disallowed commands.

These files are automatically created if they do not exist when the shell is first run. You can edit these files using the `edit-whitelist` and `edit-blacklist` commands.

## Security Features

- **Command Whitelisting**: Only commands listed in the whitelist or located in trusted directories are allowed to execute.
- **Input Sanitization**: Removes potentially harmful characters from user input to prevent injection attacks.
- **Process Isolation**: Executes commands in isolated processes to minimize the risk of system compromise.
- **Job Tracking**: Tracks background jobs and allows users to manage them.
- **Service Management**: Provides commands to start, stop, restart, and check the status of system services.

## Contributing

Contributions are welcome! If you have any suggestions, bug reports, or feature requests, please open an issue or submit a pull request.

## License

This project is licensed under the **GNU Affero General Public License (AGPL)**. See the [LICENSE](LICENSE) file for more details.

---

Enjoy using SecShell! If you have any questions or need further assistance, feel free to reach out.
