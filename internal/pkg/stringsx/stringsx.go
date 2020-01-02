package stringsx

import "strings"

// Split splits a string in two by a separator. If the separator is not
// present in s, then s and an empty string are returned.
func Split(s string, sep string) (string, string) {
	switch len(sep) {
	case 0:
		return s, ""
	case 1:
		return SplitByte(s, sep[0])
	}

	i := strings.Index(s, sep)
	if i < 0 {
		return s, ""
	}

	end := i + len(sep)
	return s[:i], s[end:]
}

// SplitByte splits a string in two by a separator. If the separator is not
// present in s, then s and an empty string are returned.
func SplitByte(s string, sep byte) (string, string) {
	i := strings.IndexByte(s, sep)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}
