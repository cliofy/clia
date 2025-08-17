# CLIA CLI - Command Line Intelligence Assistant

## Overview

CLIA is an intelligent command-line assistant that converts natural language queries to shell commands. Unlike traditional terminal replacements, CLIA operates as a standard CLI tool without entering alternate screen mode, making it perfect for automation and scripting.

## Features

- 🤖 **Natural Language Processing**: Convert plain English to shell commands
- 🛡️ **Safety First**: Built-in risk assessment for dangerous commands  
- 🎨 **Standard CLI**: Works in your existing terminal without alt-screen
- 🔌 **Multiple Providers**: Support for OpenAI, OpenRouter, and Ollama
- 📝 **Session Mode**: Interactive conversations without leaving your terminal
- ⚡ **Direct Execution**: Execute commands with PTY support for proper formatting

## Installation

```bash
# Build from source
go build -o clia ./cmd/clia

# Install to PATH
sudo mv clia /usr/local/bin/
```

## Configuration

CLIA stores configuration in `~/.clia/config.yaml`:

```yaml
active_provider: openai
providers:
  openai:
    type: openai
    config:
      api_key: ${OPENAI_API_KEY}
      model: gpt-3.5-turbo
  ollama:
    type: ollama
    config:
      base_url: http://localhost:11434
      model: llama2
```

## Usage

### Basic Commands

```bash
# Natural language query
clia "show all docker containers"

# Skip confirmation
clia -y "list large files in current directory"

# Dry run (show command without executing)
clia --dry-run "delete all temp files"

# Direct command execution (bypass AI)
clia exec "docker ps -a"
```

### Configuration Management

```bash
# Show config file path
clia config path

# Set active provider
clia config set provider ollama

# List all configuration
clia config list
```

### Provider Management

```bash
# List all providers
clia provider list

# Show active provider
clia provider active

# Switch provider
clia provider set openai

# Test provider connection
clia provider test
```

### Interactive Session

```bash
# Start interactive session (stays in main terminal)
clia session

# In session:
CLIA [openai] ~/project> find large files
# ... AI suggests command ...
Execute? (y/n/e[dit]) > y

# Direct execution in session
CLIA [openai] ~/project> !ls -la
```

### Command History

```bash
# Show recent commands
clia history

# Show last 10 commands
clia history --limit 10

# Clear history
clia history clear
```

## Architecture

CLIA follows a clean modular architecture:

```
clia/
├── cmd/clia/          # Main entry point
├── cli/              
│   ├── commands/      # Command implementations
│   └── output/        # Formatters and UI
├── core/
│   ├── agent/         # AI agent for NLP
│   ├── executor/      # PTY-based execution
│   ├── provider/      # LLM provider abstraction
│   └── config/        # Configuration management
```

## Safety Features

CLIA includes comprehensive safety checks:

- **Critical Risk**: System destruction commands (rm -rf /, mkfs, etc.)
- **High Risk**: Home directory deletion, permission changes
- **Medium Risk**: File overwrites, current directory deletion
- **Low Risk**: Sudo usage, force kills

Risk levels affect:
- Confirmation requirements
- Confidence scores
- Visual warnings

## Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
./test_cli.sh

# Test with specific provider
OPENAI_API_KEY=your-key go test -tags=integration
```

## Environment Variables

- `CLIA_CONFIG_DIR`: Override config directory location
- `OPENAI_API_KEY`: OpenAI API key
- `OPENROUTER_KEY`: OpenRouter API key
- `SHELL`: Default shell for command execution

## Non-Interactive Mode

Perfect for automation and scripting:

```bash
# In scripts
COMMAND=$(clia --dry-run "compress all log files" | grep "Suggested" -A1 | tail -1)
eval $COMMAND

# With auto-confirm
clia -y "restart nginx service"

# Pipe to other commands
clia exec "ps aux" | grep python
```

## Comparison with Alternatives

| Feature | CLIA | Warp Terminal | Fig |
|---------|------|---------------|-----|
| Natural Language | ✅ | ✅ | ✅ |
| Standard CLI | ✅ | ❌ | ❌ |
| No Alt-Screen | ✅ | ❌ | ❌ |
| Scriptable | ✅ | Limited | Limited |
| Local LLM Support | ✅ | ❌ | ❌ |
| Open Source | ✅ | ❌ | ❌ |

## Contributing

CLIA is open source and welcomes contributions:

1. Fork the repository
2. Create a feature branch
3. Write tests (TDD approach)
4. Implement the feature
5. Submit a pull request

## License

MIT License - See LICENSE file for details

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [creack/pty](https://github.com/creack/pty) for PTY management
- Powered by various LLM providers