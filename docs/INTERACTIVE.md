# CLIA Interactive Command Execution

## Overview

CLIA now supports full interactive/TUI command execution. When you run commands like `top`, `vim`, or other terminal-based programs through CLIA, they will work exactly as if you ran them directly in your terminal.

## How It Works

CLIA automatically detects when a command requires interactive mode and switches to a full PTY (pseudo-terminal) pass-through mode. This allows:

- **Full terminal control**: Keyboard input, special keys, and terminal resize events are properly handled
- **Color and formatting**: ANSI escape sequences and terminal formatting work correctly
- **TUI applications**: Programs like `top`, `htop`, `vim`, and others work as expected
- **Interactive sessions**: SSH, database clients, and REPLs function normally

## Usage Examples

### Direct Execution

```bash
# Run top with full interactivity
./clia exec top

# Edit a file with vim
./clia exec "vim README.md"

# SSH to a remote server
./clia exec "ssh user@server"

# Start an interactive Python session
./clia exec python

# Monitor logs with tail -f
./clia exec "tail -f /var/log/system.log"
```

### In Interactive Session

```bash
# Start CLIA session
./clia session

# Use ! prefix for direct execution
CLIA> !top
CLIA> !vim config.yaml
CLIA> !docker exec -it mycontainer bash
```

### Natural Language Queries

```bash
# CLIA will detect if the suggested command needs interactive mode
./clia "monitor system resources"
# Suggests: top
# Automatically uses interactive mode when executed

./clia "edit the readme file"
# Suggests: vim README.md
# Automatically uses interactive mode when executed
```

## Supported Interactive Commands

CLIA automatically detects these types of commands:

### System Monitoring
- `top`, `htop`, `btop` - Process monitors
- `watch` - Command repeater
- `tail -f` - Log following

### Text Editors
- `vim`, `vi`, `nano`, `emacs` - Text editors
- `less` - File pager

### Development Tools
- `python`, `node`, `ruby` - Language REPLs (when run without arguments)
- `mysql`, `psql`, `redis-cli`, `mongo` - Database clients
- `lazygit`, `tig` - Git interfaces

### Terminal Multiplexers
- `tmux`, `screen` - Terminal multiplexers

### Remote Access
- `ssh`, `telnet`, `ftp` - Remote connection tools

### Container Tools
- `docker exec -it` - Interactive container execution
- `docker attach` - Container attachment

### File Managers
- `mc`, `ranger`, `nnn` - Terminal file managers
- `ncdu` - Disk usage analyzer

## How Detection Works

CLIA uses several methods to detect interactive commands:

1. **Command matching**: Checks against a list of known interactive commands
2. **Flag detection**: Looks for interactive flags like `-it`, `-i -t`
3. **REPL detection**: Identifies language interpreters run without arguments

## Technical Details

### PTY Mode

When an interactive command is detected, CLIA:
1. Creates a pseudo-terminal (PTY)
2. Sets the terminal to raw mode to pass through all input
3. Handles window resize signals (SIGWINCH)
4. Bidirectionally copies data between your terminal and the PTY

### Standard Mode

For non-interactive commands, CLIA:
1. Captures output to a buffer
2. Preserves ANSI escape sequences
3. Returns the complete output after execution
4. Allows for output processing and analysis

## Troubleshooting

### Command Not Recognized as Interactive

If a command should be interactive but isn't detected:
- Check if it's in the supported commands list
- Consider using the appropriate flags (e.g., `-it` for Docker)
- Report the issue for inclusion in future updates

### Terminal Issues

If you experience terminal issues after running an interactive command:
- The terminal state is automatically restored after each command
- If issues persist, run `reset` or `stty sane` to restore terminal settings

### Performance

Interactive commands run with minimal overhead:
- Direct PTY pass-through ensures native performance
- No buffering or processing delays
- Terminal resize events are handled in real-time

## Examples

### Monitor System with top
```bash
$ ./clia exec top
ℹ Starting interactive session: top
[Full top interface appears]
# Press 'q' to exit
```

### Edit Configuration
```bash
$ ./clia exec "vim ~/.clia/config.yaml"
ℹ Starting interactive session: vim ~/.clia/config.yaml
[Vim editor opens]
# Edit and save with :wq
```

### Interactive Docker Container
```bash
$ ./clia exec "docker exec -it myapp bash"
ℹ Starting interactive session: docker exec -it myapp bash
root@container:/app# 
# Full bash session in container
```

### Python REPL
```bash
$ ./clia exec python
ℹ Starting interactive session: python
Python 3.9.7 (default, Sep  3 2021, 12:37:55)
>>> print("Hello from CLIA!")
Hello from CLIA!
>>> exit()
```

## Integration with AI

When CLIA's AI suggests a command that requires interactive mode:
1. The command is analyzed before execution
2. If interactive mode is needed, it's automatically enabled
3. After completion, the session history is updated

This seamless integration means you can use natural language to launch interactive programs:
- "edit my shell configuration" → `vim ~/.bashrc`
- "monitor network connections" → `netstat -c`
- "connect to my database" → `mysql -u root -p`

All will work with full interactivity when executed.