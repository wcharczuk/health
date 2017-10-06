// +build darwin freebsd dragonfly openbsd netbsd

package health

import "syscall"

const (
	getTermios = syscall.TIOCGETA
	setTermios = syscall.TIOCSETA
)
