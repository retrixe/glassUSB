package main

// truncateString truncates a string to a specified *byte* length. (not string!)
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	// []rune would be better for multi-byte characters, but this is for ASCII filesystem labels.
	return s[:maxLength]
}
