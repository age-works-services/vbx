package process

import "syscall"

// SignalError preserves the host cancellation signal for child forwarding and exit status.
type SignalError struct{ Signal syscall.Signal }

func (e SignalError) Error() string { return e.Signal.String() }
func (e SignalError) ExitCode() int { return 128 + int(e.Signal) }
