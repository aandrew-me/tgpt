//go:build freebsd
// +build freebsd

package clipboard

// CopyToClipboard is a no-op on FreeBSD as clipboard functionality is not supported
func CopyToClipboard(text string) {
	// Clipboard functionality not available on FreeBSD
}
