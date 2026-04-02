package services

import (
	"strings"
	"testing"
)

func TestCheckKeywordMatch(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		keywords  []string
		whitelist []string
		wantMatch bool
	}{
		{
			name:      "exact match",
			content:   "hello world",
			keywords:  []string{"hello"},
			whitelist: nil,
			wantMatch: true,
		},
		{
			name:      "case insensitive",
			content:   "HELLO world",
			keywords:  []string{"hello"},
			whitelist: nil,
			wantMatch: true,
		},
		{
			name:      "whitelist bypass",
			content:   "hello world",
			keywords:  []string{"hello"},
			whitelist: []string{"hello"},
			wantMatch: false,
		},
		{
			name:      "wildcard match",
			content:   "click here for prizes",
			keywords:  []string{"click*"},
			whitelist: nil,
			wantMatch: true,
		},
		{
			name:      "no match",
			content:   "goodbye world",
			keywords:  []string{"hello"},
			whitelist: nil,
			wantMatch: false,
		},
		{
			name:      "empty keywords",
			content:   "hello world",
			keywords:  nil,
			whitelist: nil,
			wantMatch: false,
		},
		{
			name:      "substring match",
			content:   "say hello to everyone",
			keywords:  []string{"hello"},
			whitelist: nil,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := CheckKeywordMatch(tt.content, tt.keywords, tt.whitelist)
			if got != tt.wantMatch {
				t.Errorf("CheckKeywordMatch() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestCheckRegexMatch(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		patterns  []string
		whitelist []string
		wantMatch bool
	}{
		{
			name:      "regex match",
			content:   "test123abc",
			patterns:  []string{`\d+`},
			whitelist: nil,
			wantMatch: true,
		},
		{
			name:      "no match",
			content:   "hello world",
			patterns:  []string{`\d+`},
			whitelist: nil,
			wantMatch: false,
		},
		{
			name:      "whitelist bypass",
			content:   "hello world",
			patterns:  []string{`hello`},
			whitelist: []string{"hello"},
			wantMatch: false,
		},
		{
			name:      "case insensitive",
			content:   "TEST123",
			patterns:  []string{`test`},
			whitelist: nil,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := CheckRegexMatch(tt.content, tt.patterns, tt.whitelist)
			if got != tt.wantMatch {
				t.Errorf("CheckRegexMatch() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestCheckSpamPatterns(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantSpam   bool
	}{
		{
			name:     "not spam",
			content:  "This is a normal message.",
			wantSpam: false,
		},
		{
			name:     "excessive caps",
			content:  "THIS IS ALL CAPS AND VERY LOUD AND ANNOYING",
			wantSpam: true,
		},
		{
			name:     "repeated characters",
			content:  "Hellooooooooooooooooooo there",
			wantSpam: true,
		},
		{
			name:     "spam phrase",
			content:  "Click here to claim your prize!",
			wantSpam: true,
		},
		{
			name:     "normal sentence",
			content:  "Hello, how are you today?",
			wantSpam: false,
		},
		{
			name:     "empty content",
			content:  "",
			wantSpam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckSpamPatterns(tt.content)
			if got != tt.wantSpam {
				t.Errorf("CheckSpamPatterns(%q) = %v, want %v", tt.content, got, tt.wantSpam)
			}
		})
	}
}

func TestCheckMentionAbuse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		mentionLimit int
		wantAbuse   bool
	}{
		{
			name:        "no mentions",
			content:     "Hello world",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "under limit",
			content:     "@user1 @user2 @user3",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "at limit",
			content:     "@user1 @user2 @user3 @user4 @user5",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "over limit",
			content:     "@user1 @user2 @user3 @user4 @user5 @user6",
			mentionLimit: 5,
			wantAbuse:   true,
		},
		{
			name:        "role mentions count",
			content:     "@user1 @user2 <@&123456>",
			mentionLimit: 3,
			wantAbuse:   true,
		},
		{
			name:        "zero limit",
			content:     "@user1",
			mentionLimit: 0,
			wantAbuse:   false,
		},
		{
			name:        "negative limit",
			content:     "@user1",
			mentionLimit: -1,
			wantAbuse:   false,
		},
		{
			name:        "unicode in content - no crash",
			content:     "Hello 👋🎉 @user1 🚀✨",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "emoji cluster with mention",
			content:     "🎊🎉🎈🎁🎂✨ @user1 💫💫💫",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "mixed unicode letters and mention",
			content:     "Привет @user1",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "chinese characters with mention",
			content:     "你好 @user1 你好吗",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "japanese characters with mention",
			content:     "こんにちは @user1",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "arabic characters with mention",
			content:     "مرحبا @user1",
			mentionLimit: 5,
			wantAbuse:   false,
		},
		{
			name:        "high unicode with mention abuse",
			content:     "@user1 @user2 @user3 @user4 @user5 @user6 👨‍👩‍👧‍👦🔥💫",
			mentionLimit: 5,
			wantAbuse:   true,
		},
		{
			name:        "many emoji clusters with mentions",
			content:     "@user1 @user2 @user3 @user4 @user5 @user6 🌟💥🔥💫",
			mentionLimit: 5,
			wantAbuse:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMentionAbuse(tt.content, tt.mentionLimit)
			if got != tt.wantAbuse {
				t.Errorf("CheckMentionAbuse(%q, %d) = %v, want %v", tt.content, tt.mentionLimit, got, tt.wantAbuse)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name  string
		input byte
		want  bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"special char", '@', false},
		{"newline", '\n', false},
		{"tab", '\t', false},
		{"null", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlphanumeric(tt.input)
			if got != tt.want {
				t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeForMatching(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"lowercase", "Hello", "hello"},
		{"trim spaces", "  hello  ", "hello"},
		{"mixed case", "HeLLo WoRLd", "hello world"},
		{"already normalized", "hello", "hello"},
		{"with newlines", "hello\nworld", "hello\nworld"},
		{"with tabs", "hello\tworld", "hello\tworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeForMatching(tt.input)
			if got != tt.want {
				t.Errorf("normalizeForMatching(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPregQuote(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
	}{
		{"no special chars", "hello", "hello"},
		{"dot", "hello.world", `hello\.world`},
		{"asterisk", "hello*world", `hello\*world`},
		{"question mark", "hello?world", `hello\?world`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pregQuote(tt.input)
			if got != tt.want {
				t.Errorf("pregQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCheckMentionAbuse_G115_RuneOverflow tests the specific G115 vulnerability:
// when runes > 255 are cast to byte, there is potential for overflow/bypass.
// The isAlphanumeric function receives byte only, so we test that high Unicode
// runes (e.g., emoji) don't cause incorrect word boundary detection.
func TestCheckMentionAbuse_G115_RuneOverflow(t *testing.T) {
	// G115: isAlphanumeric receives byte(r) which truncates when r > 255.
	// The check `r > 255` in the calling code should skip those runes,
	// but the isAlphanumeric function itself only sees bytes 0-255.
	// This test ensures emoji/emoji序列 don't cause crashes or incorrect counts.

	// Many emojis are > 255 but should not crash the function
	emojiHeavy := strings.Repeat("🎉", 50) + " @user1"
	got := CheckMentionAbuse(emojiHeavy, 5)
	// The emoji sequences are not mentions, so only 1 mention should be counted
	if got != false {
		t.Errorf("CheckMentionAbuse with emoji heavy content: got %v, want false", got)
	}

	// High-density Unicode with actual mentions
	highUnicode := "🏴󠁧󠁢󠁥󠁮󠁧󠁿 @user1 👨‍👩‍👧‍👦 @user2"
	got = CheckMentionAbuse(highUnicode, 5)
	if got != false {
		t.Errorf("CheckMentionAbuse with high Unicode: got %v, want false", got)
	}

	// Unicode letters (Cyrillic) with mentions - should work correctly
	cyrillic := "Привет @user1 как дела"
	got = CheckMentionAbuse(cyrillic, 5)
	if got != false {
		t.Errorf("CheckMentionAbuse with Cyrillic: got %v, want false", got)
	}
}