package memory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Storage handles reading and writing memory data to YAML files
type Storage struct {
	filePath   string
	backupDir  string
	maxBackups int
}

// NewStorage creates a new storage instance
func NewStorage(filePath string) *Storage {
	dir := filepath.Dir(filePath)
	backupDir := filepath.Join(dir, "backups")
	
	return &Storage{
		filePath:   filePath,
		backupDir:  backupDir,
		maxBackups: 5,
	}
}

// Load loads memory from the YAML file
func (s *Storage) Load() (*Memory, error) {
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// Return empty memory if file doesn't exist
		return NewMemory(), nil
	}

	// Open and read file
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory file: %w", err)
	}
	defer file.Close()

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read memory file: %w", err)
	}

	// Parse YAML
	var memory Memory
	if err := yaml.Unmarshal(data, &memory); err != nil {
		return nil, fmt.Errorf("failed to parse memory YAML: %w", err)
	}

	// Validate loaded data
	if err := s.validateMemory(&memory); err != nil {
		return nil, fmt.Errorf("memory validation failed: %w", err)
	}

	return &memory, nil
}

// Save saves memory to the YAML file
func (s *Storage) Save(memory *Memory) error {
	// Validate memory before saving
	if err := s.validateMemory(memory); err != nil {
		return fmt.Errorf("memory validation failed: %w", err)
	}

	// Create backup before saving
	if err := s.createBackup(); err != nil {
		// Log warning but don't fail the save
		fmt.Printf("Warning: Failed to create backup: %v\n", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(memory)
	if err != nil {
		return fmt.Errorf("failed to marshal memory to YAML: %w", err)
	}

	// Add header comment
	header := fmt.Sprintf("# clia memory file\n# Generated on %s\n# Total entries: %d\n\n",
		time.Now().Format(time.RFC3339),
		len(memory.Entries))
	
	content := header + string(data)

	// Write to temporary file first
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic move
	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to move temporary file: %w", err)
	}

	// Clean up old backups
	if err := s.cleanupOldBackups(); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: Failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// createBackup creates a backup of the current memory file
func (s *Storage) createBackup() error {
	// Check if source file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// No file to backup
		return nil
	}

	// Create backup directory
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("memory_%s.yaml", timestamp)
	backupPath := filepath.Join(s.backupDir, backupName)

	// Copy file
	return s.copyFile(s.filePath, backupPath)
}

// copyFile copies a file from src to dst
func (s *Storage) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// cleanupOldBackups removes old backup files beyond maxBackups
func (s *Storage) cleanupOldBackups() error {
	if _, err := os.Stat(s.backupDir); os.IsNotExist(err) {
		return nil // No backup directory
	}

	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return err
	}

	// Filter backup files
	var backupFiles []os.DirEntry
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "memory_") && strings.HasSuffix(file.Name(), ".yaml") {
			backupFiles = append(backupFiles, file)
		}
	}

	// If we have more than maxBackups, remove the oldest ones
	if len(backupFiles) <= s.maxBackups {
		return nil
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		entry   os.DirEntry
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range backupFiles {
		info, err := file.Info()
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			entry:   file,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.After(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove oldest files
	toRemove := len(fileInfos) - s.maxBackups
	for i := 0; i < toRemove; i++ {
		filePath := filepath.Join(s.backupDir, fileInfos[i].entry.Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Warning: Failed to remove old backup %s: %v\n", filePath, err)
		}
	}

	return nil
}

// validateMemory validates the memory structure
func (s *Storage) validateMemory(memory *Memory) error {
	if memory == nil {
		return fmt.Errorf("memory is nil")
	}

	// Validate metadata
	if memory.Metadata.Version == "" {
		memory.Metadata.Version = "1.0"
	}

	// Validate entries
	validEntries := make([]MemoryEntry, 0, len(memory.Entries))
	for i, entry := range memory.Entries {
		if !entry.IsValid() {
			fmt.Printf("Warning: Skipping invalid entry at index %d: %+v\n", i, entry)
			continue
		}
		validEntries = append(validEntries, entry)
	}

	memory.Entries = validEntries
	memory.Metadata.TotalEntries = len(validEntries)

	return nil
}

// GetBackupFiles returns a list of available backup files
func (s *Storage) GetBackupFiles() ([]BackupInfo, error) {
	if _, err := os.Stat(s.backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "memory_") && strings.HasSuffix(file.Name(), ".yaml") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			backup := BackupInfo{
				Filename: file.Name(),
				Path:     filepath.Join(s.backupDir, file.Name()),
				Size:     info.Size(),
				ModTime:  info.ModTime(),
			}

			backups = append(backups, backup)
		}
	}

	return backups, nil
}

// RestoreFromBackup restores memory from a backup file
func (s *Storage) RestoreFromBackup(backupPath string) error {
	// Validate backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Create backup of current file before restoring
	if err := s.createBackup(); err != nil {
		fmt.Printf("Warning: Failed to backup current file before restore: %v\n", err)
	}

	// Copy backup file to current location
	return s.copyFile(backupPath, s.filePath)
}

// GetFileInfo returns information about the memory file
func (s *Storage) GetFileInfo() (FileInfo, error) {
	info := FileInfo{
		Path:   s.filePath,
		Exists: false,
	}

	stat, err := os.Stat(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return info, err
	}

	info.Exists = true
	info.Size = stat.Size()
	info.ModTime = stat.ModTime()

	return info, nil
}

// BackupInfo represents information about a backup file
type BackupInfo struct {
	Filename string    `json:"filename"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
}

// FileInfo represents information about the memory file
type FileInfo struct {
	Path    string    `json:"path"`
	Exists  bool      `json:"exists"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}