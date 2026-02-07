//go:build !freebsd
// +build !freebsd

package clipboard

import (
	"fmt"

	"golang.design/x/clipboard"
)

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(text string) {
	err := clipboard.Init()

	if err == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
		fmt.Println("Copied command to clipboard")
	}
}
