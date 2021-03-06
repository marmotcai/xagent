// +build darwin freebsd linux netbsd openbsd

package proxy

import (
	"path"
)

const (
	rcFname      = "rc"
	blockedFname = "blocked"
	directFname  = "direct"
	statFname    = "stat"

	newLine = "\n"
)

func getDefaultRcFile() string {
	return path.Join(path.Join(getUserHomeDir(), ".cow", rcFname))
}
