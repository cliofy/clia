package memory

import (
	"sort"
	"strings"
	"unicode"
)

// Search handles searching through memory entries
type Search struct {
	// Cache for expensive operations
	keywordCache map[string][]string
}

// NewSearch creates a new search instance
func NewSearch() *Search {
	return &Search{
		keywordCache: make(map[string][]string),
	}
}

// Search searches for relevant memory entries based on a query
func (s *Search) Search(query string, entries []MemoryEntry, options SearchOptions) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	normalizedQuery := s.normalizeQuery(query)
	queryKeywords := s.extractKeywords(normalizedQuery)

	var results []SearchResult

	for _, entry := range entries {
		// Skip failed entries if not including failures
		if !options.IncludeFailures && !entry.Success {
			continue
		}

		// Calculate relevance score
		score, matchType, reason := s.calculateRelevance(normalizedQuery, queryKeywords, entry)

		// Filter by minimum score
		if score >= options.MinScore {
			results = append(results, SearchResult{
				Entry:     entry,
				Score:     score,
				Reason:    reason,
				MatchType: matchType,
			})
		}
	}

	// Sort results
	s.sortResults(results, options.SortBy)

	// Limit results
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		results = results[:options.MaxResults]
	}

	return results, nil
}

// calculateRelevance calculates the relevance score for a memory entry
func (s *Search) calculateRelevance(query string, queryKeywords []string, entry MemoryEntry) (float64, MatchType, string) {
	var maxScore float64
	var bestMatchType MatchType
	var bestReason string

	// 1. Exact match check
	if exactScore, reason := s.checkExactMatch(query, entry); exactScore > maxScore {
		maxScore = exactScore
		bestMatchType = MatchTypeExact
		bestReason = reason
	}

	// 2. Fuzzy string similarity
	if fuzzyScore, reason := s.checkFuzzyMatch(query, entry); fuzzyScore > maxScore {
		maxScore = fuzzyScore
		bestMatchType = MatchTypeFuzzy
		bestReason = reason
	}

	// 3. Keyword matching
	if keywordScore, reason := s.checkKeywordMatch(queryKeywords, entry); keywordScore > maxScore {
		maxScore = keywordScore
		bestMatchType = MatchTypeKeyword
		bestReason = reason
	}

	// 4. Command pattern matching
	if commandScore, reason := s.checkCommandMatch(query, entry); commandScore > maxScore {
		maxScore = commandScore
		bestMatchType = MatchTypeCommand
		bestReason = reason
	}

	// 5. Semantic similarity (simple)
	if semanticScore, reason := s.checkSemanticMatch(query, entry); semanticScore > maxScore {
		maxScore = semanticScore
		bestMatchType = MatchTypeSemantic
		bestReason = reason
	}

	// Apply entry relevance boost
	entryRelevance := entry.RelevanceScore()
	finalScore := maxScore * (0.7 + entryRelevance*0.3)

	return finalScore, bestMatchType, bestReason
}

// checkExactMatch checks for exact string matches
func (s *Search) checkExactMatch(query string, entry MemoryEntry) (float64, string) {
	normalizedRequest := strings.ToLower(entry.NormalizedRequest)
	
	if normalizedRequest == query {
		return 1.0, "Exact request match"
	}

	if strings.Contains(normalizedRequest, query) {
		return 0.9, "Request contains query"
	}

	if strings.Contains(query, normalizedRequest) {
		return 0.8, "Query contains request"
	}

	return 0.0, ""
}

// checkFuzzyMatch checks for fuzzy string similarity
func (s *Search) checkFuzzyMatch(query string, entry MemoryEntry) (float64, string) {
	// Calculate edit distance similarity
	requestSimilarity := s.calculateEditDistanceSimilarity(query, entry.NormalizedRequest)
	
	if requestSimilarity > 0.7 {
		return requestSimilarity, "Similar request pattern"
	}

	// Check description similarity
	if entry.Description != "" {
		descSimilarity := s.calculateEditDistanceSimilarity(query, strings.ToLower(entry.Description))
		if descSimilarity > 0.6 {
			return descSimilarity * 0.8, "Similar description"
		}
	}

	return 0.0, ""
}

// checkKeywordMatch checks for keyword-based matching
func (s *Search) checkKeywordMatch(queryKeywords []string, entry MemoryEntry) (float64, string) {
	if len(queryKeywords) == 0 {
		return 0.0, ""
	}

	entryKeywords := s.extractKeywords(entry.NormalizedRequest)
	if entry.Description != "" {
		entryKeywords = append(entryKeywords, s.extractKeywords(strings.ToLower(entry.Description))...)
	}

	// Count matching keywords
	matchCount := 0
	var matchedKeywords []string

	for _, queryKeyword := range queryKeywords {
		for _, entryKeyword := range entryKeywords {
			if queryKeyword == entryKeyword || strings.Contains(entryKeyword, queryKeyword) {
				matchCount++
				matchedKeywords = append(matchedKeywords, queryKeyword)
				break
			}
		}
	}

	if matchCount == 0 {
		return 0.0, ""
	}

	// Calculate score based on match ratio
	score := float64(matchCount) / float64(len(queryKeywords))
	reason := "Matched keywords: " + strings.Join(matchedKeywords, ", ")

	return score * 0.85, reason
}

// checkCommandMatch checks for command pattern similarity
func (s *Search) checkCommandMatch(query string, entry MemoryEntry) (float64, string) {
	// Extract command-like patterns from query
	queryCommands := s.extractCommandPatterns(query)
	entryCommands := s.extractCommandPatterns(entry.SelectedCommand)

	if len(queryCommands) == 0 || len(entryCommands) == 0 {
		return 0.0, ""
	}

	// Check for command similarities
	var matchedCommands []string
	matchCount := 0

	for _, queryCmd := range queryCommands {
		for _, entryCmd := range entryCommands {
			if queryCmd == entryCmd {
				matchCount++
				matchedCommands = append(matchedCommands, queryCmd)
				break
			}
		}
	}

	if matchCount == 0 {
		return 0.0, ""
	}

	score := float64(matchCount) / float64(max(len(queryCommands), len(entryCommands)))
	reason := "Command pattern match: " + strings.Join(matchedCommands, ", ")

	return score * 0.75, reason
}

// checkSemanticMatch checks for semantic similarity (simple implementation)
func (s *Search) checkSemanticMatch(query string, entry MemoryEntry) (float64, string) {
	// Simple semantic matching based on action words
	actionMap := map[string][]string{
		"list":     {"ls", "dir", "show", "display", "find"},
		"find":     {"search", "locate", "grep", "look"},
		"copy":     {"cp", "duplicate", "backup"},
		"move":     {"mv", "rename", "relocate"},
		"delete":   {"rm", "remove", "erase"},
		"extract":  {"unzip", "tar", "decompress"},
		"compress": {"zip", "tar", "gzip"},
		"edit":     {"vim", "nano", "modify"},
		"install":  {"apt", "yum", "brew", "pip"},
		"download": {"wget", "curl", "fetch"},
	}

	queryWords := strings.Fields(query)
	entryWords := strings.Fields(entry.NormalizedRequest + " " + entry.SelectedCommand)

	for action, synonyms := range actionMap {
		queryHasAction := false
		entryHasAction := false

		// Check if query contains action or synonyms
		for _, word := range queryWords {
			if word == action || s.containsAny(synonyms, word) {
				queryHasAction = true
				break
			}
		}

		// Check if entry contains action or synonyms
		for _, word := range entryWords {
			if word == action || s.containsAny(synonyms, word) {
				entryHasAction = true
				break
			}
		}

		if queryHasAction && entryHasAction {
			return 0.6, "Semantic action match: " + action
		}
	}

	return 0.0, ""
}

// sortResults sorts search results based on the specified criteria
func (s *Search) sortResults(results []SearchResult, sortBy SortBy) {
	switch sortBy {
	case SortByRelevance:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})
	case SortByFrequency:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Entry.UsageCount > results[j].Entry.UsageCount
		})
	case SortByRecency:
		sort.Slice(results, func(i, j int) bool {
			return results[i].Entry.Timestamp.After(results[j].Entry.Timestamp)
		})
	case SortByCombined:
		sort.Slice(results, func(i, j int) bool {
			// Combined score: relevance * 0.5 + frequency * 0.3 + recency * 0.2
			scoreI := s.calculateCombinedScore(results[i])
			scoreJ := s.calculateCombinedScore(results[j])
			return scoreI > scoreJ
		})
	}
}

// calculateCombinedScore calculates a combined score for sorting
func (s *Search) calculateCombinedScore(result SearchResult) float64 {
	relevanceScore := result.Score
	frequencyScore := float64(result.Entry.UsageCount) / 10.0
	if frequencyScore > 1.0 {
		frequencyScore = 1.0
	}
	recencyScore := result.Entry.RelevanceScore()

	return relevanceScore*0.5 + frequencyScore*0.3 + recencyScore*0.2
}

// normalizeQuery normalizes a search query
func (s *Search) normalizeQuery(query string) string {
	// Convert to lowercase and trim
	normalized := strings.ToLower(strings.TrimSpace(query))
	
	// Replace multiple spaces with single space
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// extractKeywords extracts keywords from a string
func (s *Search) extractKeywords(text string) []string {
	// Check cache first
	if keywords, exists := s.keywordCache[text]; exists {
		return keywords
	}

	// Split into words and filter
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var keywords []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
	}

	for _, word := range words {
		word = strings.ToLower(word)
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	// Cache the result
	s.keywordCache[text] = keywords
	return keywords
}

// extractCommandPatterns extracts command patterns from text
func (s *Search) extractCommandPatterns(text string) []string {
	words := strings.Fields(text)
	var commands []string

	for _, word := range words {
		// Look for common command patterns
		if s.isCommandLike(word) {
			commands = append(commands, word)
		}
	}

	return commands
}

// isCommandLike checks if a word looks like a command
func (s *Search) isCommandLike(word string) bool {
	commonCommands := map[string]bool{
		"ls": true, "cd": true, "pwd": true, "cat": true, "grep": true, "find": true,
		"cp": true, "mv": true, "rm": true, "mkdir": true, "rmdir": true, "chmod": true,
		"tar": true, "zip": true, "unzip": true, "gzip": true, "gunzip": true,
		"wget": true, "curl": true, "ssh": true, "scp": true, "rsync": true,
		"git": true, "npm": true, "pip": true, "apt": true, "yum": true, "brew": true,
		"docker": true, "kubectl": true, "vim": true, "nano": true, "emacs": true,
	}

	return commonCommands[strings.ToLower(word)]
}

// calculateEditDistanceSimilarity calculates similarity based on edit distance
func (s *Search) calculateEditDistanceSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	distance := s.editDistance(s1, s2)
	maxLen := max(len(s1), len(s2))
	
	if maxLen == 0 {
		return 0.0
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// editDistance calculates the edit distance between two strings
func (s *Search) editDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	
	// Create a matrix to store distances
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				min(matrix[i-1][j]+1, matrix[i][j-1]+1), // min of deletion and insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// containsAny checks if any of the items are contained in the target string
func (s *Search) containsAny(items []string, target string) bool {
	for _, item := range items {
		if strings.Contains(target, item) {
			return true
		}
	}
	return false
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}