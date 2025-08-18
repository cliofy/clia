module github.com/yourusername/clia

go 1.24.1

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1
	github.com/creack/pty v1.1.24
	github.com/fatih/color v1.18.0
	github.com/joho/godotenv v1.5.1
	github.com/sashabaranov/go-openai v1.41.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/term v0.34.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// Use local submodule for go-ansiterm to allow customization
replace github.com/Azure/go-ansiterm => ./third_party/go-ansiterm
