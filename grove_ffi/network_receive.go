package grove_ffi

import (
	"io"
	"net"
	"os"
)

// Select the receive policy once at process startup. Existing users keep
// net.Conn.Read unless they explicitly set GROVE_TCP_RECEIVE_MODE=poll.
var receivePolling = parseReceivePolling(os.Getenv("GROVE_TCP_RECEIVE_MODE"))

func parseReceivePolling(mode string) bool {
	switch mode {
	case "", "event":
		return false
	case "poll":
		return true
	default:
		panic("GROVE_TCP_RECEIVE_MODE must be event or poll")
	}
}

func makeReceiveReader(conn net.Conn) io.Reader {
	if receivePolling {
		return newPollingReader(conn)
	}
	return conn
}
