//go:build !linux

package grove_ffi

import (
	"io"
	"net"
)

func newPollingReader(conn net.Conn) io.Reader {
	panic("GROVE_TCP_RECEIVE_MODE=poll requires Linux")
}
