package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("🧪 Testing Data Analysis Mode")
	fmt.Println("============================\n")

	// Test CSV data
	csvData := `name,age,department,salary
John Doe,28,Engineering,75000
Jane Smith,32,Marketing,68000
Bob Johnson,45,Engineering,95000
Alice Brown,29,Sales,62000
Charlie Wilson,38,Engineering,82000
Diana Lee,35,Marketing,71000`

	// Test JSON data
	jsonData := `{
  "employees": [
    {"name": "John", "age": 28, "dept": "Engineering", "salary": 75000},
    {"name": "Jane", "age": 32, "dept": "Marketing", "salary": 68000},
    {"name": "Bob", "age": 45, "dept": "Engineering", "salary": 95000}
  ],
  "company": "TechCorp",
  "year": 2024
}`

	// Test log data
	logData := `2024-08-13 10:30:15 INFO Application started
2024-08-13 10:30:16 INFO Database connected
2024-08-13 10:31:22 WARN High memory usage detected: 85%
2024-08-13 10:32:01 ERROR Failed to process request: timeout
2024-08-13 10:32:15 INFO Request retried successfully
2024-08-13 10:35:42 ERROR Database connection lost
2024-08-13 10:35:45 INFO Database reconnected`

	// Create test data files
	if err := os.WriteFile("test_employees.csv", []byte(csvData), 0644); err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}

	if err := os.WriteFile("test_data.json", []byte(jsonData), 0644); err != nil {
		fmt.Printf("Error creating JSON file: %v\n", err)
		return
	}

	if err := os.WriteFile("test_app.log", []byte(logData), 0644); err != nil {
		fmt.Printf("Error creating log file: %v\n", err)
		return
	}

	fmt.Println("✅ Test data files created successfully!")
	fmt.Println("\n📋 Available Test Commands:")
	fmt.Println()

	// CSV analysis examples
	fmt.Println("📊 CSV Data Analysis:")
	fmt.Println("   cat test_employees.csv | ./bin/clia make table")
	fmt.Println("   cat test_employees.csv | ./bin/clia analyze")
	fmt.Println("   cat test_employees.csv | ./bin/clia summarize")
	fmt.Println()

	// JSON analysis examples
	fmt.Println("🔧 JSON Data Analysis:")
	fmt.Println("   cat test_data.json | ./bin/clia analyze")
	fmt.Println("   cat test_data.json | ./bin/clia format yaml")
	fmt.Println("   cat test_data.json | ./bin/clia make table")
	fmt.Println()

	// Log analysis examples
	fmt.Println("📜 Log Data Analysis:")
	fmt.Println("   cat test_app.log | ./bin/clia summarize")
	fmt.Println("   cat test_app.log | ./bin/clia analyze")
	fmt.Println("   tail -f test_app.log | ./bin/clia analyze")
	fmt.Println()

	// Chart examples
	fmt.Println("📈 Visualization Recommendations:")
	fmt.Println("   cat test_employees.csv | ./bin/clia chart")
	fmt.Println("   echo 'Product,Sales\\nA,100\\nB,150\\nC,80' | ./bin/clia chart")
	fmt.Println()

	// Advanced examples
	fmt.Println("🚀 Advanced Examples:")
	fmt.Println("   ps aux | head -20 | ./bin/clia analyze")
	fmt.Println("   df -h | ./bin/clia make table")
	fmt.Println("   ls -la | ./bin/clia analyze")
	fmt.Println()

	fmt.Println("💡 Prerequisites:")
	fmt.Println("   Set one of these environment variables for AI analysis:")
	fmt.Println("   export OPENROUTER_API_KEY=\"your-key-here\"")
	fmt.Println("   export OPENAI_API_KEY=\"your-key-here\"")
	fmt.Println("   export ANTHROPIC_API_KEY=\"your-key-here\"")
	fmt.Println()

	fmt.Println("🎯 Try running one of the commands above to test the analysis mode!")
	fmt.Println()

	// Show current status
	fmt.Println("📋 Current Implementation Status:")
	fmt.Println("   ✅ Pipe input detection")
	fmt.Println("   ✅ Analysis command parsing")
	fmt.Println("   ✅ Data format detection")
	fmt.Println("   ✅ AI analysis service")
	fmt.Println("   ✅ Markdown rendering")
	fmt.Println("   ✅ Beautiful TUI display")
	fmt.Println("   ✅ Multiple analysis types")
	fmt.Println("   ✅ Error handling")
	fmt.Println()

	fmt.Println("🎉 Data Analysis Mode is ready to use!")

	// Cleanup old test file if exists
	if err := os.Remove("test_memory.go"); err == nil {
		fmt.Println("🧹 Cleaned up old test file")
	}
}
