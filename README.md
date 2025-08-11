# clia - Command Line Intelligent Assistant

![CI](https://github.com/yourusername/clia/workflows/CI/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/clia)](https://goreportcard.com/report/github.com/yourusername/clia)
[![Coverage Status](https://codecov.io/gh/yourusername/clia/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/clia)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> 🚀 An intelligent command-line assistant powered by AI that translates natural language into terminal commands.

## 🌟 Features

- **Natural Language Processing**: Convert plain English to shell commands
- **Context Awareness**: Intelligent suggestions based on current directory and environment
- **Multi-LLM Support**: OpenAI, Anthropic Claude, and local Ollama models
- **Interactive TUI**: Modern terminal user interface built with Bubble Tea
- **Safe Execution**: Built-in safety checks and confirmation prompts
- **Cross-Platform**: Works on Linux, macOS, and Windows

## 🚧 Development Status

This project is currently in **Phase 0** of development. See our [Implementation Plan](docs/spec/plan.md) for detailed roadmap.

### Current Phase: Project Initialization ✅
- [x] Go module setup
- [x] Project structure
- [x] Development toolchain
- [x] CI/CD pipeline
- [x] Basic documentation

### Upcoming Phases:
- **Phase 1**: TUI Framework Integration (Weeks 2-3)
- **Phase 2**: LLM Integration (Weeks 4-5)
- **Phase 3**: Command Execution Core (Weeks 6-7)
- **Phase 4**: Context Intelligence (Weeks 8-9)

## 🛠 Development Setup

### Prerequisites

- Go 1.21 or higher
- Git

### Getting Started

```bash
# Clone the repository
git clone https://github.com/yourusername/clia.git
cd clia

# Install dependencies
go mod download

# Build the project
make build

# Run the application
./bin/clia

# Check version
./bin/clia version
```

### Development Commands

```bash
# Build for development
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Lint code
make lint

# Format code
make fmt

# Run all checks
make check

# Clean build artifacts
make clean

# Show all available commands
make help
```

## 📖 Documentation

- [Product Requirements Document](docs/spec/prd.md)
- [Technical Architecture](docs/spec/技术选型.md)
- [Implementation Plan](docs/spec/plan.md)

## 🎯 Quick Example

*Note: This is the planned interface for future phases*

```bash
# Natural language input
$ clia
> find all python files larger than 1MB

# AI-generated suggestions:
1. find . -name "*.py" -size +1M
2. find . -type f -name "*.py" -exec ls -lh {} \; | awk '$5 ~ /[0-9]+M/ {print $9}'
3. du -h $(find . -name "*.py") | grep -E '[0-9]+M'

# Select and execute with safety confirmation
```

## 🏗 Architecture

```
clia/
├── cmd/clia/           # Application entry point
├── internal/           # Private application code
│   ├── ai/            # LLM integration
│   ├── config/        # Configuration management
│   ├── context/       # Context collection
│   ├── executor/      # Command execution
│   ├── tui/           # Terminal UI components
│   └── version/       # Version information
├── pkg/               # Public library code
│   └── utils/         # Utility functions
└── docs/              # Documentation
```

## 🧪 Testing

We maintain high code quality through comprehensive testing:

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run benchmarks
make bench

# View coverage report
open coverage.html
```

## 🔧 Configuration

Future configuration will be managed through:

- `~/.config/clia/config.yaml` - Main configuration file
- Environment variables for API keys
- Command-line flags for runtime options

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `make check`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the excellent TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) for beautiful terminal styling
- The Go community for amazing tools and libraries

## 📊 Project Status

- **Current Version**: Phase 0 (Development)
- **Target Release**: v1.0.0 (Week 16)
- **Language**: Go 1.21+
- **License**: MIT
- **Maintenance**: Active Development

---

*Built with ❤️ and ☕ by the clia team*