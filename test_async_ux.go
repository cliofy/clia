package main

import (
	"fmt"
	"log"

	"github.com/yourusername/clia/pkg/memory"
)

func main() {
	// Create memory manager
	manager, err := memory.NewManager()
	if err != nil {
		log.Fatalf("Failed to create memory manager: %v", err)
	}

	// Add some test entries to memory
	testEntries := []struct {
		userRequest     string
		selectedCommand string
		description     string
	}{
		{
			userRequest:     "list directory files",
			selectedCommand: "ls -la",
			description:     "List all files with detailed information",
		},
		{
			userRequest:     "show current directory",
			selectedCommand: "pwd",
			description:     "Print current working directory",
		},
		{
			userRequest:     "check disk space",
			selectedCommand: "df -h",
			description:     "Show filesystem disk space in human readable format",
		},
		{
			userRequest:     "list processes",
			selectedCommand: "ps aux",
			description:     "Show all running processes",
		},
		{
			userRequest:     "list directory content",
			selectedCommand: "ls -l",
			description:     "List directory contents with details",
		},
	}

	fmt.Println("Populating memory with test entries...")
	for _, entry := range testEntries {
		err := manager.Add(entry.userRequest, entry.selectedCommand, entry.description, "test", true)
		if err != nil {
			log.Printf("Failed to add entry: %v", err)
		} else {
			fmt.Printf("✓ Added: %s -> %s\n", entry.userRequest, entry.selectedCommand)
		}
	}

	stats := manager.GetStats()
	fmt.Printf("\n✅ Memory populated with %d entries\n", stats["total_entries"])

	fmt.Println("\n🚀 Testing async UX improvements:")
	fmt.Println("1. Run: ./bin/clia \"list current directory files\"")
	fmt.Println("   Expected: Memory suggestions show immediately, AI processing in background")
	fmt.Println("2. Run without API key to test fallback mode")
	fmt.Println("3. Try: ./bin/clia \"show disk usage\"")
	fmt.Println("   Expected: Memory suggestions if any, then AI suggestions added")

	// Test memory search directly
	fmt.Println("\n📝 Testing memory search for 'list directory':")
	options := memory.DefaultSearchOptions()
	results, err := manager.Search("list directory", options)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d memory results:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s (score: %.2f)\n", i+1, result.Entry.SelectedCommand, result.Score)
		}
	}
}
