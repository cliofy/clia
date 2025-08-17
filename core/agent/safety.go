package agent

import (
	"regexp"
	"strings"
)

// SafetyChecker checks commands for potential security risks
type SafetyChecker struct {
	config    *Config
	patterns  []SafetyPattern
	enabled   bool
}

// SafetyPattern represents a pattern to check for dangerous commands
type SafetyPattern struct {
	Pattern     *regexp.Regexp
	Level       RiskLevel
	Description string
	Mitigation  string
}

// NewSafetyChecker creates a new safety checker
func NewSafetyChecker(config *Config) *SafetyChecker {
	sc := &SafetyChecker{
		config:  config,
		enabled: config.SafetyEnabled,
	}

	// Initialize safety patterns
	sc.initPatterns()

	return sc
}

// initPatterns initializes the dangerous command patterns
func (sc *SafetyChecker) initPatterns() {
	sc.patterns = []SafetyPattern{
		// Critical risks - system destruction
		{
			Pattern:     regexp.MustCompile(`rm\s+-rf\s+/($|\s)`),
			Level:       RiskCritical,
			Description: "This command will delete the entire root filesystem",
			Mitigation:  "Never run this command. Use specific paths instead.",
		},
		{
			Pattern:     regexp.MustCompile(`rm\s+.*\s+/\*`),
			Level:       RiskCritical,
			Description: "This command will delete everything in root directory",
			Mitigation:  "Be specific about what you want to delete.",
		},
		{
			Pattern:     regexp.MustCompile(`mkfs\.`),
			Level:       RiskCritical,
			Description: "This command will format a filesystem, destroying all data",
			Mitigation:  "Make sure you have backups and are formatting the correct device.",
		},
		{
			Pattern:     regexp.MustCompile(`dd\s+.*of=/dev/(sd|hd|nvme)`),
			Level:       RiskCritical,
			Description: "This command writes directly to a disk device",
			Mitigation:  "Double-check the device name and ensure you have backups.",
		},

		// High risks - system damage
		{
			Pattern:     regexp.MustCompile(`rm\s+-rf\s+~(/|$|\s)`),
			Level:       RiskHigh,
			Description: "This command will delete your home directory",
			Mitigation:  "Be specific about what files/directories to delete.",
		},
		{
			Pattern:     regexp.MustCompile(`chmod\s+777\s+/`),
			Level:       RiskHigh,
			Description: "This gives everyone full permissions to system directories",
			Mitigation:  "Use more restrictive permissions and target specific files.",
		},
		{
			Pattern:     regexp.MustCompile(`>\s*/etc/.*\.(conf|cfg|config)`),
			Level:       RiskHigh,
			Description: "This overwrites system configuration files",
			Mitigation:  "Use >> to append or backup the file first.",
		},
		{
			Pattern:     regexp.MustCompile(`rm\s+.*\.(bashrc|zshrc|profile)`),
			Level:       RiskHigh,
			Description: "This deletes shell configuration files",
			Mitigation:  "Backup these files before deletion.",
		},
		{
			Pattern:     regexp.MustCompile(`>\s+.*\.(bashrc|zshrc|profile|bash_profile)`),
			Level:       RiskHigh,
			Description: "This overwrites shell configuration files",
			Mitigation:  "Backup the file first or use >> to append.",
		},
		{
			Pattern:     regexp.MustCompile(`chown\s+-R\s+.*\s+/`),
			Level:       RiskHigh,
			Description: "This recursively changes ownership of system files",
			Mitigation:  "Be specific about which directories to change.",
		},

		// Medium risks - data loss
		{
			Pattern:     regexp.MustCompile(`rm\s+-rf\s+\.`),
			Level:       RiskMedium,
			Description: "This deletes the current directory and all contents",
			Mitigation:  "Make sure you're in the correct directory.",
		},
		{
			Pattern:     regexp.MustCompile(`>\s+[^>]`),
			Level:       RiskMedium,
			Description: "This overwrites a file (destructive redirect)",
			Mitigation:  "Use >> to append or check if file exists first.",
		},
		{
			Pattern:     regexp.MustCompile(`mv\s+.*\s+/dev/null`),
			Level:       RiskMedium,
			Description: "This moves files to /dev/null (permanent deletion)",
			Mitigation:  "Use rm if you want to delete, or move to trash instead.",
		},
		{
			Pattern:     regexp.MustCompile(`find\s+.*-delete`),
			Level:       RiskMedium,
			Description: "This deletes files found by find command",
			Mitigation:  "Run without -delete first to see what will be deleted.",
		},

		// Low risks - caution needed
		{
			Pattern:     regexp.MustCompile(`sudo\s+`),
			Level:       RiskLow,
			Description: "This runs with elevated privileges",
			Mitigation:  "Make sure you understand what the command does.",
		},
		{
			Pattern:     regexp.MustCompile(`kill\s+-9`),
			Level:       RiskLow,
			Description: "This forcefully kills processes",
			Mitigation:  "Try regular kill first, use -9 as last resort.",
		},
		{
			Pattern:     regexp.MustCompile(`pkill|killall`),
			Level:       RiskLow,
			Description: "This kills processes by name",
			Mitigation:  "Make sure you're killing the right processes.",
		},
	}
}

// CheckCommand checks a command for security risks
func (sc *SafetyChecker) CheckCommand(command string) []SecurityRisk {
	if !sc.enabled {
		return nil
	}

	var risks []SecurityRisk

	// Check against all patterns
	for _, pattern := range sc.patterns {
		if pattern.Pattern.MatchString(command) {
			risks = append(risks, SecurityRisk{
				Level:       pattern.Level,
				Description: pattern.Description,
				Mitigation:  pattern.Mitigation,
			})
		}
	}

	// Additional checks
	risks = append(risks, sc.checkAdditionalRisks(command)...)

	return risks
}

// checkAdditionalRisks performs additional risk checks
func (sc *SafetyChecker) checkAdditionalRisks(command string) []SecurityRisk {
	var risks []SecurityRisk

	// Check for recursive operations on root paths
	if strings.Contains(command, "-r") || strings.Contains(command, "-R") {
		if strings.Contains(command, " /") || strings.Contains(command, "/*") {
			risks = append(risks, SecurityRisk{
				Level:       RiskHigh,
				Description: "Recursive operation on system directories",
				Mitigation:  "Be specific about target directories.",
			})
		}
	}

	// Check for pipe to shell
	if regexp.MustCompile(`\|\s*(bash|sh|zsh)`).MatchString(command) {
		risks = append(risks, SecurityRisk{
			Level:       RiskMedium,
			Description: "Piping to shell can execute arbitrary commands",
			Mitigation:  "Review the input carefully before execution.",
		})
	}

	// Check for curl/wget piped to shell
	if regexp.MustCompile(`(curl|wget).*\|\s*(bash|sh)`).MatchString(command) {
		risks = append(risks, SecurityRisk{
			Level:       RiskHigh,
			Description: "Downloading and executing remote scripts",
			Mitigation:  "Download and review the script first before execution.",
		})
	}

	// Check for password in command line
	if regexp.MustCompile(`(password|passwd|pwd)=`).MatchString(strings.ToLower(command)) {
		risks = append(risks, SecurityRisk{
			Level:       RiskMedium,
			Description: "Password visible in command line",
			Mitigation:  "Use environment variables or config files for passwords.",
		})
	}

	return risks
}

// IsCommandSafe returns true if the command has no high or critical risks
func (sc *SafetyChecker) IsCommandSafe(command string) bool {
	risks := sc.CheckCommand(command)
	
	for _, risk := range risks {
		if risk.Level == RiskCritical || risk.Level == RiskHigh {
			return false
		}
	}
	
	return true
}

// GetRiskLevel returns the highest risk level for a command
func (sc *SafetyChecker) GetRiskLevel(command string) RiskLevel {
	risks := sc.CheckCommand(command)
	
	if len(risks) == 0 {
		return RiskLow
	}

	highestRisk := RiskLow
	for _, risk := range risks {
		if risk.Level == RiskCritical {
			return RiskCritical
		}
		if risk.Level == RiskHigh && highestRisk != RiskCritical {
			highestRisk = RiskHigh
		}
		if risk.Level == RiskMedium && highestRisk == RiskLow {
			highestRisk = RiskMedium
		}
	}
	
	return highestRisk
}

// ShouldConfirm returns true if the command should require user confirmation
func (sc *SafetyChecker) ShouldConfirm(command string) bool {
	level := sc.GetRiskLevel(command)
	return level == RiskHigh || level == RiskCritical
}