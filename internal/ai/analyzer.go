package ai

import (
	"context"
	"fmt"
	"strings"
)

// AnalysisType represents different types of data analysis
type AnalysisType string

const (
	AnalysisTypeTable     AnalysisType = "table"
	AnalysisTypeAnalyze   AnalysisType = "analyze"
	AnalysisTypeSummarize AnalysisType = "summarize"
	AnalysisTypeFormat    AnalysisType = "format"
	AnalysisTypeChart     AnalysisType = "chart"
)

// AnalysisRequest represents a data analysis request
type AnalysisRequest struct {
	InputData       string
	AnalysisCommand string
	AnalysisType    AnalysisType
	OutputFormat    string
}

// AnalysisResponse represents the response from data analysis
type AnalysisResponse struct {
	Result       string
	OutputFormat string
	AnalysisType AnalysisType
	Error        error
}

// parseAnalysisCommand parses the analysis command to determine type and parameters
func parseAnalysisCommand(command string) AnalysisRequest {
	command = strings.ToLower(strings.TrimSpace(command))

	var analysisType AnalysisType
	var outputFormat string = "markdown"

	// Determine analysis type based on command
	switch {
	case strings.Contains(command, "make table") || strings.Contains(command, "table"):
		analysisType = AnalysisTypeTable
	case strings.Contains(command, "analyze") || strings.Contains(command, "analysis"):
		analysisType = AnalysisTypeAnalyze
	case strings.Contains(command, "summarize") || strings.Contains(command, "summary"):
		analysisType = AnalysisTypeSummarize
	case strings.Contains(command, "format"):
		analysisType = AnalysisTypeFormat
		// Check for specific format requests
		if strings.Contains(command, "json") {
			outputFormat = "json"
		} else if strings.Contains(command, "yaml") {
			outputFormat = "yaml"
		}
	case strings.Contains(command, "chart") || strings.Contains(command, "graph"):
		analysisType = AnalysisTypeChart
	default:
		// Default to analysis if no specific type found
		analysisType = AnalysisTypeAnalyze
	}

	return AnalysisRequest{
		AnalysisCommand: command,
		AnalysisType:    analysisType,
		OutputFormat:    outputFormat,
	}
}

// AnalyzeData performs data analysis using AI
func (s *Service) AnalyzeData(ctx context.Context, inputData, analysisCommand string) (*AnalysisResponse, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("no LLM provider configured")
	}

	if !s.provider.IsConfigured() {
		return nil, fmt.Errorf("LLM provider is not properly configured")
	}

	// Parse the analysis command
	request := parseAnalysisCommand(analysisCommand)
	request.InputData = inputData

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Build analysis prompt based on type
	promptText, err := s.buildAnalysisPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build analysis prompt: %w", err)
	}

	// Create completion request
	completionReq := &CompletionRequest{
		Prompt:      promptText,
		MaxTokens:   2000, // Increase token limit for analysis results
		Temperature: 0.1,  // Lower temperature for more consistent analysis
	}

	// Call LLM provider
	response, err := s.provider.Complete(ctx, completionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis from LLM: %w", err)
	}

	// Return analysis response
	return &AnalysisResponse{
		Result:       response.Content,
		OutputFormat: request.OutputFormat,
		AnalysisType: request.AnalysisType,
		Error:        nil,
	}, nil
}

// buildAnalysisPrompt builds the appropriate prompt for the analysis type
func (s *Service) buildAnalysisPrompt(request AnalysisRequest) (string, error) {
	// Detect data format
	dataFormat := detectDataFormat(request.InputData)

	// Build prompt based on analysis type
	switch request.AnalysisType {
	case AnalysisTypeTable:
		return s.buildTablePrompt(request.InputData, dataFormat), nil
	case AnalysisTypeAnalyze:
		return s.buildAnalyzePrompt(request.InputData, dataFormat), nil
	case AnalysisTypeSummarize:
		return s.buildSummarizePrompt(request.InputData, dataFormat), nil
	case AnalysisTypeFormat:
		return s.buildFormatPrompt(request.InputData, dataFormat, request.OutputFormat), nil
	case AnalysisTypeChart:
		return s.buildChartPrompt(request.InputData, dataFormat), nil
	default:
		return s.buildAnalyzePrompt(request.InputData, dataFormat), nil
	}
}

// detectDataFormat attempts to detect the format of input data
func detectDataFormat(data string) string {
	data = strings.TrimSpace(data)

	// Check for CSV (comma-separated with header)
	lines := strings.Split(data, "\n")
	if len(lines) > 1 {
		firstLine := lines[0]
		if strings.Contains(firstLine, ",") && !strings.HasPrefix(data, "{") && !strings.HasPrefix(data, "[") {
			return "csv"
		}
	}

	// Check for JSON
	if strings.HasPrefix(data, "{") || strings.HasPrefix(data, "[") {
		return "json"
	}

	// Check for YAML
	if strings.Contains(data, ":") && (strings.Contains(data, "\n") || strings.Contains(data, "---")) {
		return "yaml"
	}

	// Check for TSV (tab-separated)
	if strings.Contains(data, "\t") {
		return "tsv"
	}

	// Check for log format (timestamp patterns)
	if strings.Contains(data, "[") && strings.Contains(data, "]") &&
		(strings.Contains(data, "INFO") || strings.Contains(data, "ERROR") || strings.Contains(data, "DEBUG")) {
		return "log"
	}

	// Default to plain text
	return "text"
}

// buildTablePrompt builds prompt for table conversion
func (s *Service) buildTablePrompt(data, format string) string {
	return fmt.Sprintf(`You are a data formatting expert. Your task is to convert the provided %s data into a clean, well-formatted markdown table.

Requirements:
1. Create a proper markdown table with headers
2. Align columns appropriately 
3. Handle missing or inconsistent data gracefully
4. Keep the table readable and well-structured
5. If the data is too large, include the first 20 rows and add a note about truncation

Input data (%s format):
%s

Please convert this data into a markdown table:`, format, format, data)
}

// buildAnalyzePrompt builds prompt for data analysis
func (s *Service) buildAnalyzePrompt(data, format string) string {
	return fmt.Sprintf(`You are a data analyst. Analyze the provided %s data and provide insights in markdown format.

Your analysis should include:
1. **Data Overview**: What type of data this is and its structure
2. **Key Statistics**: Important numbers, counts, patterns
3. **Notable Insights**: Interesting findings or anomalies 
4. **Summary**: Main takeaways in bullet points

Please format your response as markdown with clear headings and structure.

Input data (%s format):
%s

Provide your analysis:`, format, format, data)
}

// buildSummarizePrompt builds prompt for data summarization
func (s *Service) buildSummarizePrompt(data, format string) string {
	return fmt.Sprintf(`You are a data summarization expert. Create a concise summary of the provided %s data.

Your summary should include:
1. **Data Type**: Brief description of what this data represents
2. **Key Metrics**: Most important numbers or counts
3. **Main Points**: 3-5 bullet points highlighting the most important information
4. **Conclusion**: One sentence summary

Keep the summary concise but informative. Format as markdown.

Input data (%s format):
%s

Provide your summary:`, format, format, data)
}

// buildFormatPrompt builds prompt for data formatting
func (s *Service) buildFormatPrompt(data, inputFormat, outputFormat string) string {
	return fmt.Sprintf(`You are a data conversion expert. Convert the provided %s data to %s format.

Requirements:
1. Maintain all data integrity during conversion
2. Use proper %s formatting and structure
3. Handle any inconsistencies in the source data
4. Ensure the output is valid %s

Input data (%s format):
%s

Convert to %s format:`, inputFormat, outputFormat, outputFormat, outputFormat, inputFormat, data, outputFormat)
}

// buildChartPrompt builds prompt for chart/visualization suggestions
func (s *Service) buildChartPrompt(data, format string) string {
	return fmt.Sprintf(`You are a data visualization expert. Analyze the provided %s data and suggest appropriate charts or visualizations.

Your response should include:
1. **Data Suitability**: What types of charts work best for this data
2. **Recommended Visualizations**: 2-3 specific chart types with explanations
3. **Key Variables**: Which columns/fields should be used for X/Y axes
4. **Insights to Highlight**: What patterns or trends the charts should emphasize
5. **ASCII Chart** (if possible): A simple text-based visualization of the data

Format your response as markdown with clear sections.

Input data (%s format):
%s

Provide visualization recommendations:`, format, format, data)
}
