package utils

import "fmt"

// HumanizeBytes formats a byte count into a readable string.
func HumanizeBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// HumanizeBytesCompact formats a byte count to compact units without space, e.g., 1536 -> "1.50K", 2.25 GB -> "2.25G".
func HumanizeBytesCompact(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.2fT", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2fG", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2fM", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2fK", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
