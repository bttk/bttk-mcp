//go:build !windows

package googleapi

import (
	"os/exec"
	"runtime"
)

// openBrowser opens the specified URL in the default browser on non-Windows platforms.
func openBrowser(url string) error {
	var cmd string
	var args []string

	if runtime.GOOS == "darwin" {
		cmd = "open"
	} else { // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
