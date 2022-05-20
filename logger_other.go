package azugo

import (
	"os"

	"github.com/mattn/go-isatty"
)

func init() {
	// when running as a systemd unit with logging set to console, the output can not be colorized,
	// otherwise it spams the journal / syslog with escape sequences
	// this file covers non-windows platforms.
	canColorStdout = isatty.IsTerminal(os.Stdout.Fd())
}
