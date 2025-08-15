package prompt

import (
	"fmt"
	"strings"
)

// AnalysisPromptTemplate provides specialized prompts for data analysis
type AnalysisPromptTemplate struct {
	InputData    string
	DataFormat   string
	OutputFormat string
}

// NewAnalysisPromptTemplate creates a new analysis prompt template
func NewAnalysisPromptTemplate(inputData, dataFormat, outputFormat string) *AnalysisPromptTemplate {
	return &AnalysisPromptTemplate{
		InputData:    inputData,
		DataFormat:   dataFormat,
		OutputFormat: outputFormat,
	}
}

// BuildTablePrompt builds a prompt for converting data to tables
func (apt *AnalysisPromptTemplate) BuildTablePrompt() string {
	template := `You are a data formatting expert specializing in creating clean, readable markdown tables.

**Task**: Convert the provided %s data into a well-formatted markdown table.

**Requirements**:
1. Create proper markdown table syntax with headers
2. Align columns appropriately for readability
3. Handle missing or inconsistent data gracefully
4. Ensure table is clean and well-structured
5. If data has more than 20 rows, show first 20 and add truncation note
6. Include column headers if not present in source data

**Data Format**: %s
**Input Data**:
%s

**Instructions**: 
- Start your response with the markdown table
- Do not include explanations before the table
- Use proper markdown table syntax with | separators
- Ensure headers are properly separated with |---|
- Handle special characters appropriately

Please convert this data to a markdown table:`

	return fmt.Sprintf(template, apt.DataFormat, apt.DataFormat, apt.InputData)
}

// BuildAnalysisPrompt builds a prompt for comprehensive data analysis
func (apt *AnalysisPromptTemplate) BuildAnalysisPrompt() string {
	template := `You are an expert data analyst. Provide a comprehensive analysis of the provided %s data.

**Structure your analysis as follows**:

# Data Analysis Report

## Overview
- Data type and structure description
- Number of records/entries
- Key characteristics

## Key Statistics
- Important numerical summaries
- Distributions and patterns
- Notable metrics

## Insights
- Interesting findings or trends
- Anomalies or outliers
- Patterns worth highlighting

## Summary
- 3-5 key takeaways in bullet points
- Main conclusions

**Data Format**: %s
**Input Data**:
%s

Please provide your analysis in the above markdown format:`

	return fmt.Sprintf(template, apt.DataFormat, apt.DataFormat, apt.InputData)
}

// BuildSummaryPrompt builds a prompt for data summarization
func (apt *AnalysisPromptTemplate) BuildSummaryPrompt() string {
	template := `You are a data summarization specialist. Create a concise but informative summary of the provided %s data.

**Format your summary as**:

# Data Summary

## What This Data Shows
Brief description of the data type and content

## Key Numbers
- Most important metrics and counts
- Critical values or ranges

## Main Points
- 3-5 bullet points with key information
- Focus on actionable insights

## Quick Conclusion
One-sentence summary of the most important finding

**Data Format**: %s
**Input Data**:
%s

Keep the summary concise but ensure it captures the essential information:`

	return fmt.Sprintf(template, apt.DataFormat, apt.DataFormat, apt.InputData)
}

// BuildFormatPrompt builds a prompt for data format conversion
func (apt *AnalysisPromptTemplate) BuildFormatPrompt() string {
	template := `You are a data conversion expert. Convert the provided %s data to %s format.

**Conversion Requirements**:
1. Maintain complete data integrity
2. Use proper %s formatting standards
3. Handle any data inconsistencies gracefully
4. Ensure output is valid and well-structured %s
5. Preserve all meaningful information

**Source Format**: %s
**Target Format**: %s
**Input Data**:
%s

**Instructions**:
- Provide only the converted data in your response
- Do not include explanations or comments
- Ensure the output is properly formatted %s
- Handle edge cases appropriately

Convert the data to %s format:`

	return fmt.Sprintf(template, apt.DataFormat, apt.OutputFormat, apt.OutputFormat,
		apt.OutputFormat, apt.DataFormat, apt.OutputFormat, apt.InputData,
		apt.OutputFormat, apt.OutputFormat)
}

// BuildChartPrompt builds a prompt for visualization recommendations
func (apt *AnalysisPromptTemplate) BuildChartPrompt() string {
	template := `You are a data visualization expert. Analyze the provided %s data and recommend appropriate visualizations.

**Provide recommendations in this format**:

# Visualization Recommendations

## Data Assessment
- What type of data this is
- Variables suitable for visualization
- Data characteristics relevant to charting

## Recommended Charts
For each recommendation, include:
- **Chart Type**: Specific chart name
- **Best For**: What insights it reveals
- **Variables**: Which data columns to use
- **Why**: Justification for this choice

## Implementation Notes
- Key considerations for creating these charts
- Potential challenges or limitations
- Suggestions for chart styling or grouping

## ASCII Preview (if possible)
Create a simple text-based representation of the most suitable chart

**Data Format**: %s
**Input Data**:
%s

Provide your visualization analysis and recommendations:`

	return fmt.Sprintf(template, apt.DataFormat, apt.DataFormat, apt.InputData)
}

// GetDataFormatDescription returns a description of common data formats
func GetDataFormatDescription(format string) string {
	descriptions := map[string]string{
		"csv":  "Comma-separated values with optional headers",
		"json": "JavaScript Object Notation with nested structures",
		"yaml": "YAML format with key-value pairs and indentation",
		"tsv":  "Tab-separated values format",
		"log":  "Application log format with timestamps and levels",
		"text": "Plain text or unstructured data",
		"xml":  "Extensible Markup Language with tags",
	}

	if desc, exists := descriptions[format]; exists {
		return desc
	}
	return "Unknown or custom data format"
}

// ValidatePrompt performs basic validation on analysis prompts
func ValidatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	if len(prompt) < 50 {
		return fmt.Errorf("prompt too short, may not provide enough context")
	}

	if len(prompt) > 8000 {
		return fmt.Errorf("prompt too long, may exceed model limits")
	}

	return nil
}

// OptimizePromptForModel adjusts prompt based on model capabilities
func OptimizePromptForModel(prompt, modelName string) string {
	// For models with shorter context windows, provide more focused prompts
	if strings.Contains(strings.ToLower(modelName), "gpt-3.5") {
		// Add instruction for concise output
		prompt += "\n\nNote: Please keep your response concise and focused on the most important information."
	}

	// For code-focused models, emphasize structured output
	if strings.Contains(strings.ToLower(modelName), "code") {
		prompt += "\n\nPlease structure your response with clear sections and markdown formatting."
	}

	return prompt
}
