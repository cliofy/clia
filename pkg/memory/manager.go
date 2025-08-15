package memory

import (
	"crypto/sha256"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/clia/pkg/utils"
)

// Manager handles memory operations with thread safety
type Manager struct {
	memory     *Memory
	config     MemoryConfig
	memoryFile string
	mutex      sync.RWMutex
	storage    *Storage
	search     *Search
}

// NewManager creates a new memory manager
func NewManager() (*Manager, error) {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	memoryFile := filepath.Join(configDir, "memory.yaml")
	config := DefaultMemoryConfig()

	storage := NewStorage(memoryFile)
	search := NewSearch()

	manager := &Manager{
		memory:     NewMemory(),
		config:     config,
		memoryFile: memoryFile,
		storage:    storage,
		search:     search,
	}

	// Try to load existing memory
	if err := manager.Load(); err != nil {
		log.Printf("Warning: Could not load memory from %s: %v", memoryFile, err)
		// Continue with empty memory
	}

	return manager, nil
}

// NewManagerWithConfig creates a memory manager with custom configuration
func NewManagerWithConfig(config MemoryConfig, memoryFile string) (*Manager, error) {
	storage := NewStorage(memoryFile)
	search := NewSearch()

	manager := &Manager{
		memory:     NewMemory(),
		config:     config,
		memoryFile: memoryFile,
		storage:    storage,
		search:     search,
	}

	if err := manager.Load(); err != nil {
		log.Printf("Warning: Could not load memory from %s: %v", memoryFile, err)
	}

	return manager, nil
}

// Load loads memory from file
func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	memory, err := m.storage.Load()
	if err != nil {
		return err
	}

	m.memory = memory
	return nil
}

// Save saves memory to file
func (m *Manager) Save() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.memory.Metadata.LastUpdated = time.Now()
	m.memory.Metadata.TotalEntries = len(m.memory.Entries)

	return m.storage.Save(m.memory)
}

// Search searches for relevant memory entries
func (m *Manager) Search(query string, options SearchOptions) ([]SearchResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.search.Search(query, m.memory.Entries, options)
}

// Add adds a new memory entry
func (m *Manager) Add(userRequest, selectedCommand, description, source string, success bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Normalize the request
	normalizedRequest := m.normalizeRequest(userRequest)

	// Check if similar entry already exists
	existingEntry := m.findSimilarEntry(normalizedRequest, selectedCommand)
	if existingEntry != nil {
		// Update existing entry
		existingEntry.UsageCount++
		existingEntry.Timestamp = time.Now()
		existingEntry.Success = success
		if description != "" {
			existingEntry.Description = description
		}
	} else {
		// Create new entry
		entry := MemoryEntry{
			ID:                uuid.New().String(),
			UserRequest:       userRequest,
			NormalizedRequest: normalizedRequest,
			SelectedCommand:   selectedCommand,
			Description:       description,
			Success:           success,
			Timestamp:         time.Now(),
			UsageCount:        1,
			Source:            source,
		}

		if !entry.IsValid() {
			return fmt.Errorf("invalid memory entry: %+v", entry)
		}

		m.memory.Entries = append(m.memory.Entries, entry)
	}

	// Cleanup if necessary
	if len(m.memory.Entries) > m.config.MaxEntries {
		m.cleanup()
	}

	// Auto-save
	go func() {
		if err := m.Save(); err != nil {
			log.Printf("Warning: Failed to save memory: %v", err)
		}
	}()

	return nil
}

// Remove removes a memory entry by ID
func (m *Manager) Remove(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, entry := range m.memory.Entries {
		if entry.ID == id {
			// Remove entry
			m.memory.Entries = append(m.memory.Entries[:i], m.memory.Entries[i+1:]...)

			// Auto-save
			go func() {
				if err := m.Save(); err != nil {
					log.Printf("Warning: Failed to save memory after removal: %v", err)
				}
			}()

			return nil
		}
	}

	return fmt.Errorf("memory entry with ID %s not found", id)
}

// Update updates an existing memory entry
func (m *Manager) Update(id string, updates map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, entry := range m.memory.Entries {
		if entry.ID == id {
			// Apply updates
			if val, ok := updates["user_request"].(string); ok {
				entry.UserRequest = val
				entry.NormalizedRequest = m.normalizeRequest(val)
			}
			if val, ok := updates["selected_command"].(string); ok {
				entry.SelectedCommand = val
			}
			if val, ok := updates["description"].(string); ok {
				entry.Description = val
			}
			if val, ok := updates["success"].(bool); ok {
				entry.Success = val
			}
			if val, ok := updates["source"].(string); ok {
				entry.Source = val
			}

			// Update timestamp
			entry.Timestamp = time.Now()
			m.memory.Entries[i] = entry

			// Auto-save
			go func() {
				if err := m.Save(); err != nil {
					log.Printf("Warning: Failed to save memory after update: %v", err)
				}
			}()

			return nil
		}
	}

	return fmt.Errorf("memory entry with ID %s not found", id)
}

// GetAll returns all memory entries
func (m *Manager) GetAll() []MemoryEntry {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid external modification
	entries := make([]MemoryEntry, len(m.memory.Entries))
	copy(entries, m.memory.Entries)
	return entries
}

// GetStats returns memory statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	totalUsage := 0
	successCount := 0
	for _, entry := range m.memory.Entries {
		totalUsage += entry.UsageCount
		if entry.Success {
			successCount++
		}
	}

	successRate := 0.0
	if len(m.memory.Entries) > 0 {
		successRate = float64(successCount) / float64(len(m.memory.Entries))
	}

	return map[string]interface{}{
		"total_entries": len(m.memory.Entries),
		"total_usage":   totalUsage,
		"success_count": successCount,
		"success_rate":  successRate,
		"last_updated":  m.memory.Metadata.LastUpdated,
		"memory_file":   m.memoryFile,
	}
}

// Cleanup removes old and low-usage entries
func (m *Manager) Cleanup() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cleanup()

	// Save after cleanup
	return m.storage.Save(m.memory)
}

// cleanup performs internal cleanup (requires lock)
func (m *Manager) cleanup() {
	var keepEntries []MemoryEntry

	now := time.Now()
	for _, entry := range m.memory.Entries {
		// Keep entry if it meets retention criteria
		if entry.UsageCount >= m.config.MinUsageCount &&
			now.Sub(entry.Timestamp) <= m.config.MaxAge {
			keepEntries = append(keepEntries, entry)
		}
	}

	// Sort by relevance score and keep top entries
	sort.Slice(keepEntries, func(i, j int) bool {
		return keepEntries[i].RelevanceScore() > keepEntries[j].RelevanceScore()
	})

	// Limit to max entries
	if len(keepEntries) > m.config.MaxEntries {
		keepEntries = keepEntries[:m.config.MaxEntries]
	}

	log.Printf("Memory cleanup: %d -> %d entries", len(m.memory.Entries), len(keepEntries))
	m.memory.Entries = keepEntries
}

// Export exports memory to a file
func (m *Manager) Export(filePath string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	storage := NewStorage(filePath)
	return storage.Save(m.memory)
}

// Import imports memory from a file
func (m *Manager) Import(filePath string, merge bool) error {
	storage := NewStorage(filePath)
	importedMemory, err := storage.Load()
	if err != nil {
		return fmt.Errorf("failed to load import file: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if merge {
		// Merge with existing memory
		for _, entry := range importedMemory.Entries {
			// Check for duplicates
			existing := m.findSimilarEntry(entry.NormalizedRequest, entry.SelectedCommand)
			if existing != nil {
				// Update usage count and timestamp
				existing.UsageCount += entry.UsageCount
				if entry.Timestamp.After(existing.Timestamp) {
					existing.Timestamp = entry.Timestamp
				}
			} else {
				// Add new entry
				m.memory.Entries = append(m.memory.Entries, entry)
			}
		}
	} else {
		// Replace existing memory
		m.memory = importedMemory
	}

	// Cleanup if necessary
	if len(m.memory.Entries) > m.config.MaxEntries {
		m.cleanup()
	}

	return m.storage.Save(m.memory)
}

// normalizeRequest normalizes a user request for better matching
func (m *Manager) normalizeRequest(request string) string {
	// Convert to lowercase
	normalized := strings.ToLower(request)

	// Remove extra whitespace
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")

	return normalized
}

// findSimilarEntry finds an existing similar entry
func (m *Manager) findSimilarEntry(normalizedRequest, command string) *MemoryEntry {
	for i := range m.memory.Entries {
		entry := &m.memory.Entries[i]

		// Check for exact normalized request match or command match
		if entry.NormalizedRequest == normalizedRequest ||
			entry.SelectedCommand == command {
			return entry
		}

		// Check for high similarity
		if m.calculateSimilarity(entry.NormalizedRequest, normalizedRequest) > 0.9 {
			return entry
		}
	}

	return nil
}

// calculateSimilarity calculates similarity between two strings (simple implementation)
func (m *Manager) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Simple hash-based similarity
	h1 := sha256.Sum256([]byte(s1))
	h2 := sha256.Sum256([]byte(s2))

	same := 0
	for i := 0; i < len(h1); i++ {
		if h1[i] == h2[i] {
			same++
		}
	}

	return float64(same) / float64(len(h1))
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() MemoryConfig {
	return m.config
}

// SetConfig updates the configuration
func (m *Manager) SetConfig(config MemoryConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = config
}
