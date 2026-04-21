package grove_ffi

import (
	"net"
	"time"
)

// UdpSocket represents a connectionless UDP socket
type udpSocket struct {
	conn    *net.UDPConn
	timeout time.Duration // read timeout (0 means no timeout)
}

type UdpSocket *udpSocket

// UdpListen creates a UDP socket bound to the given address
func UdpListen(host Address) UdpSocket {
	addr := &net.UDPAddr{
		IP:   parseIP(host),
		Port: int(parsePort(host)),
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		// Assume() no error on Listen
		panic(err)
	}
	return &udpSocket{conn: conn}
}

// UdpSend sends data to the specified remote address
// Returns true on error, false on success
func UdpSend(sock UdpSocket, remote_addr Address, data []byte) bool {
	// UDP packet size limit
	if len(data) > 65535 {
		return true // error: packet too large
	}

	addr := &net.UDPAddr{
		IP:   parseIP(remote_addr),
		Port: int(parsePort(remote_addr)),
	}

	_, err := sock.conn.WriteToUDP(data, addr)
	return err != nil
}

// UdpReceiveRet is the return type for UdpReceive
type UdpReceiveRet struct {
	Err        bool
	SenderAddr Address
	Data       []byte
}

// UdpReceive receives a datagram from any sender
// Returns the sender's address along with the data
func UdpReceive(sock UdpSocket) UdpReceiveRet {
	// Set read deadline if timeout is configured
	if sock.timeout > 0 {
		sock.conn.SetReadDeadline(time.Now().Add(sock.timeout))
	}

	// Allocate maximum UDP packet size
	buf := make([]byte, 65535)
	n, addr, err := sock.conn.ReadFromUDP(buf)

	if err != nil {
		return UdpReceiveRet{Err: true, SenderAddr: 0, Data: nil}
	}

	sender := addrToAddress(addr)
	return UdpReceiveRet{
		Err:        false,
		SenderAddr: sender,
		Data:       buf[:n],
	}
}

// UdpReceiveInto receives a datagram into a pre-allocated buffer.
// Returns (n, err) where n is the number of bytes read.
// On error (timeout or other), returns (0, true).
// The caller must provide a buffer large enough for the expected datagram.
func UdpReceiveInto(sock UdpSocket, buf []byte) (uint64, bool) {
	// Set read deadline if timeout is configured
	if sock.timeout > 0 {
		sock.conn.SetReadDeadline(time.Now().Add(sock.timeout))
	}

	n, _, err := sock.conn.ReadFromUDP(buf)
	if err != nil {
		return 0, true
	}
	return uint64(n), false
}

// UdpConfigTimeout configures the read timeout for the socket
// duration is in milliseconds (0 means no timeout, block forever)
func UdpConfigTimeout(sock UdpSocket, duration uint64) {
	if duration == 0 {
		sock.timeout = 0
	} else {
		sock.timeout = time.Duration(duration) * time.Millisecond
	}
}

// Helper functions to convert between Address and net.UDPAddr

func parseIP(addr Address) net.IP {
	a0 := byte(addr & 0xff)
	a1 := byte((addr >> 8) & 0xff)
	a2 := byte((addr >> 16) & 0xff)
	a3 := byte((addr >> 24) & 0xff)
	return net.IPv4(a0, a1, a2, a3)
}

func parsePort(addr Address) uint16 {
	return uint16((addr >> 32) & 0xffff)
}

func addrToAddress(addr *net.UDPAddr) Address {
	ip := addr.IP.To4()
	if ip == nil {
		// Not IPv4, return 0 (could handle IPv6 differently if needed)
		return 0
	}
	port := uint64(addr.Port)
	return uint64(ip[0]) | uint64(ip[1])<<8 | uint64(ip[2])<<16 | uint64(ip[3])<<24 | port<<32
}
