package memory

import (
	"time"
)

// MemoryEntry represents a single memory record of user interaction
type MemoryEntry struct {
	ID                string    `yaml:"id" json:"id"`
	UserRequest       string    `yaml:"user_request" json:"user_request"`
	NormalizedRequest string    `yaml:"normalized_request" json:"normalized_request"`
	SelectedCommand   string    `yaml:"selected_command" json:"selected_command"`
	Description       string    `yaml:"description" json:"description"`
	Success           bool      `yaml:"success" json:"success"`
	Timestamp         time.Time `yaml:"timestamp" json:"timestamp"`
	UsageCount        int       `yaml:"usage_count" json:"usage_count"`
	Source            string    `yaml:"source" json:"source"` // "ai", "fallback", "manual"
}

// Memory represents the complete memory structure
type Memory struct {
	Entries  []MemoryEntry `yaml:"entries" json:"entries"`
	Metadata Metadata      `yaml:"metadata" json:"metadata"`
}

// Metadata contains memory file metadata
type Metadata struct {
	Version      string    `yaml:"version" json:"version"`
	LastUpdated  time.Time `yaml:"last_updated" json:"last_updated"`
	TotalEntries int       `yaml:"total_entries" json:"total_entries"`
	MaxEntries   int       `yaml:"max_entries" json:"max_entries"`
}

// SearchResult represents a memory search result with relevance score
type SearchResult struct {
	Entry     MemoryEntry `json:"entry"`
	Score     float64     `json:"score"`      // Relevance score (0.0 - 1.0)
	Reason    string      `json:"reason"`     // Why this result was matched
	MatchType MatchType   `json:"match_type"` // Type of match
}

// MatchType represents the type of match found
type MatchType string

const (
	MatchTypeExact    MatchType = "exact"    // Exact string match
	MatchTypeFuzzy    MatchType = "fuzzy"    // Fuzzy string match
	MatchTypeKeyword  MatchType = "keyword"  // Keyword-based match
	MatchTypeSemantic MatchType = "semantic" // Semantic similarity
	MatchTypeCommand  MatchType = "command"  // Command pattern match
)

// SearchOptions represents options for memory search
type SearchOptions struct {
	MaxResults      int     `json:"max_results"`      // Maximum number of results to return
	MinScore        float64 `json:"min_score"`        // Minimum relevance score
	IncludeFailures bool    `json:"include_failures"` // Include failed commands
	SortBy          SortBy  `json:"sort_by"`          // Sort order
}

// SortBy represents sorting options for search results
type SortBy string

const (
	SortByRelevance SortBy = "relevance" // Sort by relevance score
	SortByFrequency SortBy = "frequency" // Sort by usage count
	SortByRecency   SortBy = "recency"   // Sort by timestamp
	SortByCombined  SortBy = "combined"  // Combined score
)

// MemoryConfig represents configuration for memory management
type MemoryConfig struct {
	MaxEntries        int           `yaml:"max_entries" json:"max_entries"`               // Maximum number of entries to keep
	MaxFileSize       int64         `yaml:"max_file_size" json:"max_file_size"`           // Maximum file size in bytes
	CleanupInterval   time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`     // How often to cleanup old entries
	MinUsageCount     int           `yaml:"min_usage_count" json:"min_usage_count"`       // Minimum usage count to keep entry
	MaxAge            time.Duration `yaml:"max_age" json:"max_age"`                       // Maximum age for entries
	BackupCount       int           `yaml:"backup_count" json:"backup_count"`             // Number of backup files to keep
	EnableCompression bool          `yaml:"enable_compression" json:"enable_compression"` // Enable gzip compression
}

// DefaultMemoryConfig returns the default configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxEntries:        1000,
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		CleanupInterval:   24 * time.Hour,
		MinUsageCount:     1,
		MaxAge:            90 * 24 * time.Hour, // 90 days
		BackupCount:       3,
		EnableCompression: false,
	}
}

// DefaultSearchOptions returns the default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults:      10,
		MinScore:        0.3,
		IncludeFailures: false,
		SortBy:          SortByCombined,
	}
}

// NewMemory creates a new empty memory structure
func NewMemory() *Memory {
	return &Memory{
		Entries: make([]MemoryEntry, 0),
		Metadata: Metadata{
			Version:      "1.0",
			LastUpdated:  time.Now(),
			TotalEntries: 0,
			MaxEntries:   DefaultMemoryConfig().MaxEntries,
		},
	}
}

// IsValid validates a memory entry
func (e *MemoryEntry) IsValid() bool {
	return e.ID != "" &&
		e.UserRequest != "" &&
		e.SelectedCommand != "" &&
		!e.Timestamp.IsZero()
}

// Age returns the age of the memory entry
func (e *MemoryEntry) Age() time.Duration {
	return time.Since(e.Timestamp)
}

// RelevanceScore calculates a combined relevance score based on usage and recency
func (e *MemoryEntry) RelevanceScore() float64 {
	// Base score from usage count (logarithmic scale)
	var usageScore float64
	if e.UsageCount > 0 {
		usageScore = float64(e.UsageCount) / 10.0
		if usageScore > 1.0 {
			usageScore = 1.0
		}
	}

	// Recency score (newer entries get higher scores)
	age := e.Age()
	recencyScore := 1.0
	if age > 0 {
		days := age.Hours() / 24
		recencyScore = 1.0 / (1.0 + days/30.0) // Decay over 30 days
	}

	// Success boost
	successBoost := 1.0
	if e.Success {
		successBoost = 1.2
	}

	return (usageScore*0.4 + recencyScore*0.6) * successBoost
}

// String returns a string representation of the memory entry
func (e *MemoryEntry) String() string {
	return e.UserRequest + " -> " + e.SelectedCommand
}
