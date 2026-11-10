package grove_ffi

import (
	"io"
	"net"
	"runtime"
	"syscall"
)

type pollingReader struct {
	raw syscall.RawConn
}

func newPollingReader(conn net.Conn) io.Reader {
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		panic("TCP receive polling requires a TCP connection")
	}
	raw, err := tcp.SyscallConn()
	if err != nil {
		panic(err)
	}
	return pollingReader{raw: raw}
}

// Read keeps the receiving goroutine runnable when the socket has no data.
// RawConn.Read retains the descriptor lifetime/read lock and deadline checks.
// Returning true from its callback avoids a network-poller wait on EAGAIN;
// Gosched lets other RPCs and protocol goroutines progress, including at P=1.
// Receive still serializes complete frames and closes on framing/read errors.
func (r pollingReader) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	for {
		var n int
		var readErr error
		err := r.raw.Read(func(fd uintptr) bool {
			n, readErr = syscall.Read(int(fd), b)
			return true
		})
		if err != nil {
			return 0, err
		}
		if readErr == syscall.EINTR {
			continue
		}
		if readErr == syscall.EAGAIN || readErr == syscall.EWOULDBLOCK {
			runtime.Gosched()
			continue
		}
		if readErr != nil {
			return 0, readErr
		}
		if n == 0 {
			return 0, io.EOF
		}
		return n, nil
	}
}
