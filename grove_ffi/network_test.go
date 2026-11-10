package grove_ffi

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"runtime"
	"testing"
	"time"
)

func pair(t *testing.T) (Connection, net.Conn) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	c, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	s, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(); s.Close() })
	connection := makeConnection(c)
	_, ordinaryReader := connection.reader.(net.Conn)
	if ordinaryReader == receivePolling {
		t.Fatal("connection reader does not match the configured receive policy")
	}
	return connection, s
}

func TestReceiveModeConfiguration(t *testing.T) {
	for mode, want := range map[string]bool{"": false, "event": false, "poll": true} {
		if got := parseReceivePolling(mode); got != want {
			t.Fatalf("mode %q: got %v, want %v", mode, got, want)
		}
	}
	t.Run("reject unknown mode", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("invalid receive policy accepted")
			}
		}()
		parseReceivePolling("typo")
	})
}

func frame(data []byte) []byte {
	b := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint64(b, uint64(len(data)))
	copy(b[8:], data)
	return b
}

func boundedReceive(t *testing.T, c Connection) ReceiveRet {
	t.Helper()
	done := make(chan ReceiveRet, 1)
	go func() { done <- Receive(c) }()
	select {
	case result := <-done:
		return result
	case <-time.After(3 * time.Second):
		c.conn.Close()
		t.Fatal("Receive failed to finish")
		return ReceiveRet{}
	}
}

func TestFragmentedAndCoalescedFrames(t *testing.T) {
	c, peer := pair(t)
	payloads := [][]byte{nil, []byte("small"), bytes.Repeat([]byte{0xa5}, 4096)}
	done := make(chan error, 1)
	go func() {
		var stream []byte
		for _, data := range payloads {
			stream = append(stream, frame(data)...)
		}
		for _, length := range []int{1, 3, 5, 2, 13, len(stream)} {
			if length > len(stream) {
				length = len(stream)
			}
			if _, err := peer.Write(stream[:length]); err != nil {
				done <- err
				return
			}
			stream = stream[length:]
			time.Sleep(time.Millisecond)
		}
		done <- nil
	}()
	for _, want := range payloads {
		got := boundedReceive(t, c)
		if got.Err || !bytes.Equal(got.Data, want) {
			t.Fatalf("frame mismatch: err=%v len=%d", got.Err, len(got.Data))
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTruncatedFramesFailAndClose(t *testing.T) {
	for _, length := range []int{0, 1, 7, 8, 10} {
		c, peer := pair(t)
		b := frame([]byte("payload"))
		if _, err := peer.Write(b[:length]); err != nil {
			t.Fatal(err)
		}
		peer.Close()
		if !boundedReceive(t, c).Err {
			t.Fatalf("accepted truncated frame of %d bytes", length)
		}
		if !Send(c, []byte("after error")) {
			t.Fatal("Receive error did not close connection")
		}
	}
}

func TestCloseUnblocksReceive(t *testing.T) {
	c, _ := pair(t)
	time.AfterFunc(10*time.Millisecond, func() { c.conn.Close() })
	if !boundedReceive(t, c).Err {
		t.Fatal("closed connection accepted")
	}
}

func TestReadDeadline(t *testing.T) {
	c, _ := pair(t)
	if err := c.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if !boundedReceive(t, c).Err {
		t.Fatal("deadline did not produce error")
	}
}

func TestMultipleConnectionsProgressAtOneP(t *testing.T) {
	old := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(old)
	results := make(chan ReceiveRet, 4)
	for i := 0; i < 4; i++ {
		c, peer := pair(t)
		go func() { results <- Receive(c) }()
		go func() { time.Sleep(5 * time.Millisecond); peer.Write(frame([]byte("ok"))) }()
	}
	for i := 0; i < 4; i++ {
		select {
		case got := <-results:
			if got.Err || string(got.Data) != "ok" {
				t.Fatal("bad reply")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("polling starved another RPC")
		}
	}
}

func TestSendWireFormat(t *testing.T) {
	c, peer := pair(t)
	payload := bytes.Repeat([]byte("a"), 4096)
	done := make(chan bool, 1)
	go func() { done <- Send(c, payload) }()
	want := frame(payload)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(peer, got); err != nil {
		t.Fatal(err)
	}
	if <-done || !bytes.Equal(want, got) {
		t.Fatal("wire format changed")
	}
}

func TestConcurrentReceiversPreserveFrameBoundaries(t *testing.T) {
	c, peer := pair(t)
	results := make(chan ReceiveRet, 2)
	go func() { results <- Receive(c) }()
	go func() { results <- Receive(c) }()
	want := map[string]bool{"first": true, "second": true}
	stream := append(frame([]byte("first")), frame([]byte("second"))...)
	if _, err := peer.Write(stream); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			if result.Err || !want[string(result.Data)] {
				t.Fatalf("interleaved or repeated frame: %+v", result)
			}
			delete(want, string(result.Data))
		case <-time.After(3 * time.Second):
			t.Fatal("concurrent Receive did not finish")
		}
	}
}
