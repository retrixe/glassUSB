package main

import "strings"

// truncateString truncates a string to a specified *byte* length. (not string!)
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	// []rune would be better for multi-byte characters, but this is for ASCII filesystem labels.
	return s[:maxLength]
}

// sanitizeNTFSLabel ensures the label is valid for NTFS filesystems.
func sanitizeNTFSLabel(label string) string {
	// Yeah, this could be a lot more efficient, but who cares
	label = strings.ReplaceAll(label, "\t", "_")
	label = strings.ReplaceAll(label, ".", "_")

	return truncateString(label, 32)
}

// sanitizeFATLabel ensures the label is valid for FAT32 filesystems.
func sanitizeFATLabel(label string) string {
	return strings.ToUpper(sanitizeExFATLabel(label)) // exFAT sanitation + uppercase for FAT32
}

// sanitizeExFATLabel ensures the label is valid for exFAT filesystems.
func sanitizeExFATLabel(label string) string {
	// Yeah, this could be a lot more efficient, but who cares
	label = strings.ReplaceAll(label, "\t", "_")
	label = strings.ReplaceAll(label, ".", "_")

	// exFAT doesn't seem to support most of these either:
	// https://learn.microsoft.com/en-us/windows/win32/fileio/exfat-specification#table-35-invalid-filename-characters
	// Not sure about NTFS, but Rufus doesn't seem to think so
	// https://github.com/pbatard/rufus/blob/6d8fbf98305ff37eb531c45cbd6ff44563c53917/src/format.c#L263
	const unauthorized = "*?,;:/\\|+=<>[]\""
	for _, char := range unauthorized {
		label = strings.ReplaceAll(label, string(char), "_")
	}

	// Not necessary on exFAT, but do it anyway.
	for _, char := range label {
		if char < 0x20 || char > 0x7e { // 0x7f is DEL; 0x80+ (Extended ASCII) is prohibited on FAT32
			label = strings.ReplaceAll(label, string(char), "_")
		}
	}

	return truncateString(label, 11)
}
