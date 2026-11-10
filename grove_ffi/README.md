# TCP receive policy

Normal builds support two process-wide TCP receive policies. Set
`GROVE_TCP_RECEIVE_MODE` before starting a binary:

- `event` (also the default when unset): use `net.Conn.Read` and Go's network
  poller while waiting for data.
- `poll` (Linux): use nonblocking `syscall.Read` through `RawConn.Read`; yield
  with `runtime.Gosched` and retry on EAGAIN without parking in the network
  poller. The policy applies to accepted and outgoing TCP connections.

The flag is read once at package initialization and each connection stores its
selected reader. There is no per-request configuration lookup or additional
polling goroutine. UDP behavior is unchanged. Unsupported flag values fail at
startup; enabling polling on other operating systems fails when a connection
is constructed. The default event-driven path remains available there.

`Receive` still uses its receive mutex and `io.ReadFull` to decode the 8-byte
length header and payload. Framing, public types/signatures, Send, descriptor
lifetime protection, read deadlines, EOF handling and closing on read errors
are preserved by the implementation. Polling keeps existing receive goroutines
runnable and increases CPU consumption; it does not disable NIC interrupts or
bypass TCP. High connection counts require separate performance evaluation.

This modifies the trusted networking implementation. Matching public APIs and
passing tests do not establish that the existing FFI proof contract is
preserved. Review framing/ownership, descriptor lifetime and concurrent close,
deadline/error semantics, and scheduler/progress assumptions before adopting
the change in a verified system. No proof changes or new axioms are included.

Run the real TCP contract tests under both policies:

```sh
GROVE_TCP_RECEIVE_MODE=event go test -race -count=1 ./grove_ffi
GROVE_TCP_RECEIVE_MODE=poll go test -race -count=1 ./grove_ffi
```

The implementation follows the single-client LazyLog receive-polling experiment
documented in [LazyLog PR #9](https://github.com/SysVerification/lazylog-go/pull/9).
That experiment measured 263 → 179 µs median append latency against polling
C++ at 181 µs, with the same protocol logic and hardware. Those measurements
apply to the recorded prototype; this production integration is validated
separately and does not claim a new distributed performance measurement.
