package validation

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// SQL injection patterns
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\bunion\b.*\bselect\b)`),
		regexp.MustCompile(`(?i)(\binsert\b.*\binto\b)`),
		regexp.MustCompile(`(?i)(\bupdate\b.*\bset\b)`),
		regexp.MustCompile(`(?i)(\bdelete\b.*\bfrom\b)`),
		regexp.MustCompile(`(?i)(\bdrop\b.*\btable\b)`),
		regexp.MustCompile(`(?i)(\bexec\b|\bxp_cmdshell\b)`),
		regexp.MustCompile(`(?i)(\bwaitfor\b.*\bdelay\b)`),
		regexp.MustCompile(`--`), // SQL comment
		regexp.MustCompile(`;`),  // Statement separator
	}
	
	// XSS patterns
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`<script.*?>.*?</script>`),
		regexp.MustCompile(`javascript:`),
		regexp.MustCompile(`on\w+\s*=`),
		regexp.MustCompile(`data:`),
	}
	
	// Path traversal patterns
	pathTraversalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\./`),
		regexp.MustCompile(`\.\.\\`),
		regexp.MustCompile(`/etc/passwd`),
		regexp.MustCompile(`C:\\`),
	}
)

// ValidateInput checks for common injection attacks
func ValidateInput(input string) (bool, string) {
	// Check length
	if utf8.RuneCountInString(input) > 1000 {
		return false, "Input too long (max 1000 characters)"
	}
	
	// Check for SQL injection
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return false, "Invalid input detected"
		}
	}
	
	// Check for XSS
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return false, "Invalid input detected"
		}
	}
	
	// Check for path traversal
	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(input) {
			return false, "Invalid input detected"
		}
	}
	
	return true, ""
}

// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
	// Remove control characters
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Keep tab, LF, CR
			return -1
		}
		return r
	}, input)
	
	// Escape HTML
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	
	// Limit length
	if utf8.RuneCountInString(input) > 1000 {
		input = string([]rune(input)[:1000])
	}
	
	return input
}

// ValidateCommand validates game command input
func ValidateCommand(command string, args []string) (bool, string) {
	// Validate command itself
	if valid, msg := ValidateInput(command); !valid {
		return false, msg
	}
	
	// Validate each argument
	for _, arg := range args {
		if valid, msg := ValidateInput(arg); !valid {
			return false, msg
		}
	}
	
	return true, ""
}