package grove_ffi

import (
	"net"
	"sync"
)

// UdpSocket represents a connectionless UDP socket
type udpSocket struct {
	conn *net.UDPConn
	mu   *sync.Mutex // guards sending and receiving
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
	return &udpSocket{conn: conn, mu: new(sync.Mutex)}
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

	sock.mu.Lock()
	defer sock.mu.Unlock()

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
	sock.mu.Lock()
	defer sock.mu.Unlock()

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
