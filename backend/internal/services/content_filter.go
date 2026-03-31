package services

import (
	"regexp"
	"strings"
)

// CheckKeywordMatch checks if content matches any of the keywords.
// Returns (matched, matchedKeyword). Keyword matching is case-insensitive.
// Whitelist words are exempt from matching.
func CheckKeywordMatch(content string, keywords, whitelist []string) (bool, string) {
	if len(keywords) == 0 {
		return false, ""
	}

	normalizedContent := normalizeForMatching(content)

	// Build whitelist set for O(1) lookup
	whitelistSet := make(map[string]bool)
	for _, w := range whitelist {
		whitelistSet[normalizeForMatching(w)] = true
	}

	for _, keyword := range keywords {
		normalizedKeyword := normalizeForMatching(keyword)

		// Skip if keyword is in whitelist
		if whitelistSet[normalizedKeyword] {
			continue
		}

		// Check for exact keyword match (word boundary)
		pattern := `\b` + pregQuote(normalizedKeyword) + `\b`
		matched, _ := regexp.MatchString(pattern, normalizedContent)
		if matched {
			return true, keyword
		}

		// Check for wildcard pattern (* matches any characters)
		if strings.Contains(keyword, "*") {
			wildcardPattern := strings.ReplaceAll(normalizedKeyword, "*", ".*")
			matched, _ := regexp.MatchString("(?i)"+wildcardPattern, normalizedContent)
			if matched {
				return true, keyword
			}
		}

		// Check for substring match
		if strings.Contains(normalizedContent, normalizedKeyword) {
			return true, keyword
		}
	}

	return false, ""
}

// CheckRegexMatch checks if content matches any of the regex patterns.
// Returns (matched, matchedPattern). Matching is case-insensitive.
func CheckRegexMatch(content string, patterns, whitelist []string) (bool, string) {
	if len(patterns) == 0 {
		return false, ""
	}

	// Build whitelist set
	whitelistSet := make(map[string]bool)
	for _, w := range whitelist {
		whitelistSet[normalizeForMatching(w)] = true
	}

	for _, pattern := range patterns {
		normalizedPattern := normalizeForMatching(pattern)

		// Skip if pattern is in whitelist
		if whitelistSet[normalizedPattern] {
			continue
		}

		// Compile and execute regex (case-insensitive)
		re, err := regexp.Compile("(?i)" + normalizedPattern)
		if err != nil {
			continue
		}

		if re.MatchString(content) {
			return true, pattern
		}
	}

	return false, ""
}

// CheckSpamPatterns checks if content appears spammy.
func CheckSpamPatterns(content string) bool {
	if len(content) == 0 {
		return false
	}

	// Check for excessive caps
	capsCount := 0
	letterCount := 0
	for _, r := range content {
		if r >= 'A' && r <= 'Z' {
			capsCount++
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			letterCount++
		}
	}
	if letterCount > 10 && float64(capsCount)/float64(letterCount) > 0.7 {
		return true
	}

	// Check for repeated characters
	repeated := regexp.MustCompile(`(.)\1{9,}`)
	if repeated.MatchString(content) {
		return true
	}

	// Check for repeated words
	normalizedContent := strings.ToLower(content)
	words := strings.Fields(normalizedContent)
	if len(words) >= 3 {
		wordCount := make(map[string]int)
		for _, w := range words {
			wordCount[w]++
		}
		for _, count := range wordCount {
			if count >= 3 && len(words) <= 6 {
				return true
			}
			if count >= 5 {
				return true
			}
		}
	}

	// Check for spam phrases
	spamPhrases := []string{
		`click here`, `click this`, `act now`, `limited time`,
		`winner`, `you won`, `congratulations`, `free money`, `make money fast`,
	}
	for _, phrase := range spamPhrases {
		if strings.Contains(normalizedContent, phrase) {
			return true
		}
	}

	return false
}

// CheckMentionAbuse checks if content has too many mentions.
func CheckMentionAbuse(content string, mentionLimit int) bool {
	if mentionLimit <= 0 {
		return false
	}

	mentionCount := 0
	inWord := false
	for _, r := range content {
		if r == '@' && !inWord {
			mentionCount++
			inWord = true
		} else if r > 255 || !isAlphanumeric(byte(r)) {
			inWord = false
		}
	}

	roleMentionCount := strings.Count(content, "<@&")
	return mentionCount+roleMentionCount > mentionLimit
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func normalizeForMatching(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func pregQuote(s string) string {
	return regexp.QuoteMeta(s)
}
