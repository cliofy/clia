package version

import "runtime"

var (
	// Version will be set during build using -ldflags
	Version = "dev"
	
	// GitCommit will be set during build using -ldflags  
	GitCommit = "unknown"
	
	// BuildTime will be set during build using -ldflags
	BuildTime = "unknown"
	
	// GoVersion contains the current Go version
	GoVersion = runtime.Version()
)

// Info represents version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// GetInfo returns version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
	}
}