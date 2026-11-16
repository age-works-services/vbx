package provision

import "strings"

// lines emits a genuinely empty file for an empty requirement or module list.
func lines(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "\n") + "\n"
}
