package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMemoryEntry tests the MemoryEntry struct and its methods
func TestMemoryEntry(t *testing.T) {
	entry := MemoryEntry{
		ID:                "test-id",
		UserRequest:       "list files",
		NormalizedRequest: "list files",
		SelectedCommand:   "ls -la",
		Description:       "List all files with details",
		Success:           true,
		Timestamp:         time.Now(),
		UsageCount:        1,
		Source:            "ai",
	}

	// Test IsValid
	if !entry.IsValid() {
		t.Error("Valid entry should return true for IsValid()")
	}

	// Test invalid entry
	invalidEntry := MemoryEntry{}
	if invalidEntry.IsValid() {
		t.Error("Invalid entry should return false for IsValid()")
	}

	// Test Age
	age := entry.Age()
	if age < 0 {
		t.Error("Age should be non-negative")
	}

	// Test RelevanceScore
	score := entry.RelevanceScore()
	if score < 0 || score > 2.0 { // Max is 1.2 with success boost
		t.Errorf("RelevanceScore should be between 0 and 2.0, got %f", score)
	}

	// Test String representation
	str := entry.String()
	expected := "list files -> ls -la"
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}

// TestNewMemory tests memory creation
func TestNewMemory(t *testing.T) {
	memory := NewMemory()
	
	if memory == nil {
		t.Fatal("NewMemory should not return nil")
	}

	if len(memory.Entries) != 0 {
		t.Error("New memory should have no entries")
	}

	if memory.Metadata.Version != "1.0" {
		t.Error("New memory should have version 1.0")
	}
}

// TestDefaultConfigs tests default configurations
func TestDefaultConfigs(t *testing.T) {
	config := DefaultMemoryConfig()
	if config.MaxEntries <= 0 {
		t.Error("MaxEntries should be positive")
	}

	options := DefaultSearchOptions()
	if options.MaxResults <= 0 {
		t.Error("MaxResults should be positive")
	}
	if options.MinScore < 0 || options.MinScore > 1 {
		t.Error("MinScore should be between 0 and 1")
	}
}

// TestStorage tests YAML storage functionality
func TestStorage(t *testing.T) {
	// Create temporary file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_memory.yaml")
	
	storage := NewStorage(tempFile)

	// Test loading non-existent file (should return empty memory)
	memory, err := storage.Load()
	if err != nil {
		t.Fatalf("Loading non-existent file should not error: %v", err)
	}
	if len(memory.Entries) != 0 {
		t.Error("Loading non-existent file should return empty memory")
	}

	// Add some test data
	testEntry := MemoryEntry{
		ID:                "test-1",
		UserRequest:       "test request",
		NormalizedRequest: "test request",
		SelectedCommand:   "echo test",
		Description:       "Test command",
		Success:           true,
		Timestamp:         time.Now(),
		UsageCount:        1,
		Source:            "test",
	}
	memory.Entries = append(memory.Entries, testEntry)

	// Test saving
	err = storage.Save(memory)
	if err != nil {
		t.Fatalf("Failed to save memory: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("Memory file should exist after saving")
	}

	// Test loading saved file
	loadedMemory, err := storage.Load()
	if err != nil {
		t.Fatalf("Failed to load saved memory: %v", err)
	}

	if len(loadedMemory.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(loadedMemory.Entries))
	}

	if loadedMemory.Entries[0].ID != testEntry.ID {
		t.Error("Loaded entry ID does not match saved entry")
	}
}

// TestManager tests the memory manager
func TestManager(t *testing.T) {
	// Create temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_memory.yaml")
	
	config := DefaultMemoryConfig()
	config.MaxEntries = 5 // Small limit for testing

	manager, err := NewManagerWithConfig(config, tempFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test adding entries
	err = manager.Add("list files", "ls -la", "List files", "test", true)
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}

	// Test getting all entries
	entries := manager.GetAll()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Test adding duplicate (should update existing)
	err = manager.Add("list files", "ls -la", "Updated description", "test", true)
	if err != nil {
		t.Fatalf("Failed to add duplicate entry: %v", err)
	}

	entries = manager.GetAll()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after duplicate, got %d", len(entries))
	}

	if entries[0].UsageCount != 2 {
		t.Errorf("Expected usage count 2, got %d", entries[0].UsageCount)
	}

	// Test search
	results, err := manager.Search("list", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	// Test stats
	stats := manager.GetStats()
	if stats["total_entries"].(int) != 1 {
		t.Error("Stats should show 1 total entry")
	}

	// Test removal
	entryID := entries[0].ID
	err = manager.Remove(entryID)
	if err != nil {
		t.Fatalf("Failed to remove entry: %v", err)
	}

	entries = manager.GetAll()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after removal, got %d", len(entries))
	}
}

// TestSearch tests the search functionality
func TestSearch(t *testing.T) {
	search := NewSearch()

	// Create test entries
	entries := []MemoryEntry{
		{
			ID:                "1",
			UserRequest:       "list files",
			NormalizedRequest: "list files",
			SelectedCommand:   "ls -la",
			Description:       "List all files",
			Success:           true,
			Timestamp:         time.Now(),
			UsageCount:        5,
		},
		{
			ID:                "2",
			UserRequest:       "find large files",
			NormalizedRequest: "find large files",
			SelectedCommand:   "find . -size +100M",
			Description:       "Find files larger than 100MB",
			Success:           true,
			Timestamp:         time.Now().Add(-time.Hour),
			UsageCount:        2,
		},
		{
			ID:                "3",
			UserRequest:       "copy file",
			NormalizedRequest: "copy file",
			SelectedCommand:   "cp source dest",
			Description:       "Copy a file",
			Success:           false,
			Timestamp:         time.Now().Add(-2 * time.Hour),
			UsageCount:        1,
		},
	}

	options := DefaultSearchOptions()

	// Test exact match
	results, err := search.Search("list files", entries, options)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("Should find exact match")
	}
	if results[0].Entry.ID != "1" {
		t.Error("Should return the exact match entry")
	}

	// Test fuzzy match
	results, err = search.Search("list file", entries, options) // Missing 's'
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("Should find fuzzy match")
	}

	// Test keyword match
	results, err = search.Search("find big", entries, options)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("Should find keyword match")
	}

	// Test filtering failures
	options.IncludeFailures = false
	results, err = search.Search("copy", entries, options)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	for _, result := range results {
		if !result.Entry.Success {
			t.Error("Should not include failed entries when IncludeFailures is false")
		}
	}

	// Test including failures
	options.IncludeFailures = true
	results, err = search.Search("copy", entries, options)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	found := false
	for _, result := range results {
		if result.Entry.ID == "3" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should include failed entries when IncludeFailures is true")
	}
}

// TestSearchAlgorithms tests specific search algorithms
func TestSearchAlgorithms(t *testing.T) {
	search := NewSearch()

	// Test edit distance
	similarity := search.calculateEditDistanceSimilarity("hello", "helo")
	if similarity < 0.7 { // Should be high similarity
		t.Errorf("Edit distance similarity too low: %f", similarity)
	}

	similarity = search.calculateEditDistanceSimilarity("hello", "world")
	if similarity > 0.3 { // Should be low similarity
		t.Errorf("Edit distance similarity too high: %f", similarity)
	}

	// Test keyword extraction
	keywords := search.extractKeywords("list all files in directory")
	expectedKeywords := []string{"list", "all", "files", "directory"}
	if len(keywords) != len(expectedKeywords) {
		t.Errorf("Expected %d keywords, got %d", len(expectedKeywords), len(keywords))
	}

	// Test command detection
	commands := search.extractCommandPatterns("use ls command to list files")
	if len(commands) == 0 {
		t.Error("Should detect 'ls' as a command")
	}
	if commands[0] != "ls" {
		t.Errorf("Expected 'ls', got '%s'", commands[0])
	}
}

// TestManagerCleanup tests the cleanup functionality
func TestManagerCleanup(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_memory.yaml")
	
	config := DefaultMemoryConfig()
	config.MaxEntries = 3
	config.MinUsageCount = 2

	manager, err := NewManagerWithConfig(config, tempFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add entries with different usage counts
	entries := []struct {
		request, command string
		count            int
	}{
		{"request1", "command1", 5},
		{"request2", "command2", 3},
		{"request3", "command3", 1}, // Should be cleaned up
		{"request4", "command4", 4},
		{"request5", "command5", 1}, // Should be cleaned up
	}

	for _, entry := range entries {
		for i := 0; i < entry.count; i++ {
			err = manager.Add(entry.request, entry.command, "desc", "test", true)
			if err != nil {
				t.Fatalf("Failed to add entry: %v", err)
			}
		}
	}

	// Force cleanup
	err = manager.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Check remaining entries
	remainingEntries := manager.GetAll()
	if len(remainingEntries) > config.MaxEntries {
		t.Errorf("Expected at most %d entries after cleanup, got %d", 
			config.MaxEntries, len(remainingEntries))
	}

	// Verify low-usage entries were removed
	for _, entry := range remainingEntries {
		if entry.UsageCount < config.MinUsageCount {
			t.Errorf("Entry with usage count %d should have been cleaned up", 
				entry.UsageCount)
		}
	}
}

// TestStorageBackup tests backup functionality
func TestStorageBackup(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_memory.yaml")
	
	storage := NewStorage(tempFile)

	// Create initial memory
	memory := NewMemory()
	testEntry := MemoryEntry{
		ID:                "backup-test",
		UserRequest:       "backup test",
		NormalizedRequest: "backup test",
		SelectedCommand:   "echo backup",
		Description:       "Backup test",
		Success:           true,
		Timestamp:         time.Now(),
		UsageCount:        1,
		Source:            "test",
	}
	memory.Entries = append(memory.Entries, testEntry)

	// Save (should create backup)
	err := storage.Save(memory)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Modify and save again (should create another backup)
	memory.Entries[0].UsageCount = 2
	err = storage.Save(memory)
	if err != nil {
		t.Fatalf("Failed to save second time: %v", err)
	}

	// Check backup files
	backups, err := storage.GetBackupFiles()
	if err != nil {
		t.Fatalf("Failed to get backup files: %v", err)
	}

	if len(backups) == 0 {
		t.Error("Should have created backup files")
	}

	// Test file info
	info, err := storage.GetFileInfo()
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if !info.Exists {
		t.Error("Memory file should exist")
	}
	if info.Size == 0 {
		t.Error("Memory file should not be empty")
	}
}

// BenchmarkSearch benchmarks the search functionality
func BenchmarkSearch(b *testing.B) {
	search := NewSearch()
	
	// Create many test entries
	entries := make([]MemoryEntry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = MemoryEntry{
			ID:                string(rune(i)),
			UserRequest:       "test request " + string(rune(i)),
			NormalizedRequest: "test request " + string(rune(i)),
			SelectedCommand:   "test command " + string(rune(i)),
			Description:       "Test description " + string(rune(i)),
			Success:           true,
			Timestamp:         time.Now(),
			UsageCount:        i % 10,
		}
	}

	options := DefaultSearchOptions()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := search.Search("test request", entries, options)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}