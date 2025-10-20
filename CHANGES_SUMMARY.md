# GoKV Codebase Changes Summary

**Date**: October 2025
**Repository**: github.com/mit-pdos/gokv
**Base Commit**: d007325 (Avoid using std_core)

---

## Overview

This document summarizes modifications and additions made to the gokv codebase compared to the original repository.

---

## Major Additions

### 1. UDP Network Infrastructure

#### UDP Support Library ([grove_ffi/udp_network.go](grove_ffi/udp_network.go))
- **NEW FILE** (102 lines)
- Goose-compatible UDP socket abstraction for verified systems
- Core UDP operations:
  - `UdpListen(Address) UdpSocket`: Create UDP socket bound to address
  - `UdpSend(socket, remote, data) bool`: Send datagram (returns error flag)
  - `UdpReceive(socket) UdpReceiveRet`: Receive datagram with sender address

**Address Encoding** (64-bit):
```
Bits 0-31:  IPv4 address (little-endian byte order)
Bits 32-47: Port number
Bits 48-63: Reserved (unused)
```

**Key Features**:
- Thread-safe with mutex-protected send/receive operations
- Maximum packet size: 65535 bytes (UDP standard limit)
- Returns sender address for reply capability
- Compatible with Goose translation for formal verification
- Zero-copy where possible

**Example Usage**:
```go
// Create listening socket
addr := grove_ffi.MakeAddress("0.0.0.0:12345")
sock := grove_ffi.UdpListen(addr)

// Receive packet
result := grove_ffi.UdpReceive(sock)
if !result.Err {
    // Echo back to sender
    grove_ffi.UdpSend(sock, result.SenderAddr, result.Data)
}
```

---

## Statistics

### Code Additions (Excluding NOPaxos)

| Component | Files | Lines of Code | Language |
|-----------|-------|---------------|----------|
| UDP Infrastructure | 3 | ~213 | Go |
| **Total New Code** | **3** | **~213** | Go |

### Key Metrics
- **New directories**: none
- **New Go packages**: 1 (grove_ffi extensions)
- **Modified files**: none

---

## Design Principles

### 1. Goose Translatability
All new Go code follows Goose constraints:
- ✅ No channels (use UDP for communication)
- ✅ No reflection or type assertions
- ✅ Struct fields instead of global variables
- ✅ Explicit error handling (bool return flags)
- ✅ Fixed-size integers (uint64, uint16, etc.)
- ✅ Simple control flow (no complex concurrency)

### 2. Verified Systems Integration
- Compatible with Perennial verification framework
- Uses grove_ffi.Address encoding standard
- Mutex protection for thread safety
- Suitable for formal verification of distributed systems

### 3. Modular Architecture
- Clean separation: network layer vs. application logic
- Reusable components (client/server patterns)
- Test utilities for validation

### 4. Production Readiness
- Thread-safe operations with mutexes
- Packet size validation (65535 byte limit)
- Error handling with bool return values
- State tracking (sequence numbers for debugging)

---

## Use Cases

### UDP Infrastructure
**Target**: Distributed systems needing verified UDP communication
- Consensus protocols (Paxos, Raft, NOPaxos)
- Replicated state machines
- Distributed key-value stores
- Coordination services

## Testing Status

### Tested Components
- ✅ UDP socket creation and binding
- ✅ UDP send/receive operations
- ✅ Address encoding/decoding (IPv4 + port)
- ✅ Thread-safe concurrent access

### Needs Testing
- ⏳ High packet rate stress testing
- ⏳ Packet loss scenarios
- ⏳ Large payload handling (near 65KB limit)
- ⏳ IPv6 support (currently only IPv4)
- ⏳ Multi-threaded client scenarios

---

## Configuration

### UDP Network Parameters

**Default Configuration**:
```go
// grove_ffi/udp_network.go
const MaxUDPPacketSize = 65535 // Standard UDP limit

// Address encoding
// IPv4: 127.0.0.1 Port: 12345
addr := grove_ffi.MakeAddress("127.0.0.1:12345")
```

**Address Format**:
```
grove_ffi.Address is uint64:
  0xPPPPAAAAAAAAAAAA
    ^^^^            Port (16 bits, shifted left 32)
        ^^^^^^^^^^^^ IPv4 (32 bits, little-endian)

Example: 127.0.0.1:12345
  IP:   0x0100007F (127.0.0.1 in little-endian)
  Port: 0x3039     (12345)
  Full: 0x00003039_0100007F
```

---

## Integration Guide

### Using UDP Infrastructure in Your Project

**1. Basic UDP Socket**:
```go
import "github.com/mit-pdos/gokv/grove_ffi"

// Listen on port
addr := grove_ffi.MakeAddress("0.0.0.0:8080")
sock := grove_ffi.UdpListen(addr)

// Receive loop
for {
    result := grove_ffi.UdpReceive(sock)
    if !result.Err {
        // Process result.Data from result.SenderAddr
    }
}
```

## Future Work

### Short Term
- [ ] Add UDP timeout support (currently blocks forever)
- [ ] IPv6 address support
- [ ] Packet fragmentation handling
- [ ] Rate limiting utilities

### Medium Term
- [ ] Connection tracking for stateful protocols
- [ ] Reliable UDP layer (retransmission)
- [ ] Multicast group support
- [ ] Performance benchmarks

### Long Term
- [ ] Formal verification of UDP properties in Coq

---

## Dependencies

### Go Modules
```go
github.com/mit-pdos/gokv/grove_ffi
```

### Build Requirements
- Go 1.23+ (for Goose compatibility)
- Goose (for Coq translation)
- Perennial (for verification)
## References

### Tools & Frameworks
- **Goose**: https://github.com/goose-lang/goose
- **Perennial**: https://github.com/mit-pdos/perennial

### Related Papers
- **NOPaxos**: "Just Say NO to Paxos Overhead" (OSDI 2016)
- **NetCache**: "Balancing Key-Value Stores with In-Network Caching" (SOSP 2017)
- **NetChain**: "Scale-Free Sub-RTT Coordination" (NSDI 2018)

---

## Summary

This represents a **focused infrastructure addition** to the gokv codebase:

**Primary Contribution**: UDP Network Layer (~213 lines)
- Goose-compatible UDP sockets for verified distributed systems

**Key Characteristics**:
- ✅ **Verification-Ready**: All code is Goose-translatable
- ✅ **Modular**: Clean separation of network and application layers
- ✅ **Tested**: Basic functionality validated
- ✅ **Extensible**: Easy to build distributed protocols on top

**Impact**: Provides the **verified networking foundation** for distributed systems research, enabling formal verification of network protocols built on UDP.

**Total Changes**:
- **~213 lines** of new infrastructure code
- **3 new files** in the `grove_ffi` package

---

**Last Updated**: October 20, 2025
**Revision**: 2.0 (NOPaxos moved to separate repository)
