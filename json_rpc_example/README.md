# JSON-RPC over STDIO Example

This example demonstrates secure communication between a Go parent process (Alice) and a Deno TypeScript child process (Bob) using **JSON-RPC 2.0 over STDIO** (stdin/stdout).

## Overview

JSON-RPC over STDIO provides a simple, secure, and cross-platform inter-process communication (IPC) mechanism that leverages OS-level process isolation for security without requiring additional cryptographic protocols like TLS.

**Architecture:**

1. Alice (Go) spawns Bob (Deno) as a child process
2. JSON-RPC messages flow bidirectionally over stdin/stdout pipes
3. Both parties can act as client and server simultaneously
4. OS kernel provides secure channels between parent and child
5. No network sockets, no encryption overhead, no cross-platform compatibility issues

## Why JSON-RPC over STDIO?

### ✅ Advantages

1. **Native OS Security** - Process stdio pipes are secured by the OS kernel
   - Only parent process has access to child's stdin/stdout
   - No external processes can eavesdrop or inject messages
   - OS handles all access control and isolation

2. **Cross-Platform** - Works identically on Windows, macOS, and Linux
   - No dependency on Unix sockets (not available on Windows)
   - No need for named pipes platform-specific code
   - Standard stdin/stdout available everywhere

3. **Simple Protocol** - JSON-RPC 2.0 is well-specified and widely supported
   - Clear request/response model
   - Built-in error handling
   - Support for notifications (one-way messages)
   - Bidirectional communication

4. **No Encryption Needed** - OS provides secure channel
   - No TLS certificate management
   - No key exchange required
   - No crypto library dependencies
   - Lower CPU overhead

5. **Clean Process Model** - Child process lifecycle tied to parent
   - Automatic cleanup when parent exits
   - Simple process supervision
   - No orphaned processes or stale sockets

### ❌ Limitations

1. **Single Child Process** - One stdio connection per child
   - Cannot multiplex multiple connections over one stdin/stdout pair
   - For multiple workers, spawn multiple child processes

2. **No Network Access** - Only works for parent-child processes
   - Cannot be used for distributed systems
   - Limited to single machine IPC

3. **Buffering Considerations** - Line-based or length-prefixed framing required
   - Must handle message boundaries correctly
   - Potential for head-of-line blocking in high-throughput scenarios

## Security Properties

### ✅ Protected Against

1. **Network eavesdropping** - No network involved, all communication through OS pipes
2. **External process snooping** - OS kernel enforces exclusive access to stdio pipes
3. **Port scanning attacks** - No listening TCP/UDP ports
4. **Man-in-the-middle attacks** - OS guarantees pipe endpoints
5. **Replay attacks** - Process lifecycle ensures fresh session per execution

### ⚠️ Same Threat Model as Unix Sockets

Like Unix sockets, JSON-RPC over STDIO is vulnerable to:

1. **Local attacker with root/admin access** - Can inspect process memory, attach debuggers
2. **Compromised operating system** - Can intercept stdio pipes
3. **Process memory inspection** - Sensitive data visible in process memory

**Key Insight:** If an attacker has the privileges to compromise stdio pipes, they already have complete control of the system. The OS provides the security boundary.

### Comparison: When to Use What?

| **Feature**                    | **JSON-RPC/STDIO** | **Unix Socket** | **TCP + mTLS** | **Custom ECDH** |
| ------------------------------ | ------------------ | --------------- | -------------- | --------------- |
| Cross-platform (Win/Mac/Linux) | ✅ Yes             | ❌ No (Windows) | ✅ Yes         | ✅ Yes          |
| Setup complexity               | ✅ Low             | ✅ Low          | ⚠️ Medium       | ❌ High         |
| Performance                    | ✅ High            | ✅ High         | ⚠️ Medium       | ⚠️ Medium        |
| OS-level security              | ✅ Yes             | ✅ Yes          | ⚠️ Partial      | ⚠️ Partial       |
| Multiple connections           | ❌ No              | ✅ Yes          | ✅ Yes         | ✅ Yes          |
| Network transparency           | ❌ No              | ❌ No           | ✅ Yes         | ✅ Yes          |
| Certificate management         | ✅ None            | ✅ None         | ❌ Complex     | ⚠️ Key exchange  |
| Maintenance burden             | ✅ Low             | ✅ Low          | ⚠️ Medium       | ❌ High         |

## Implementation Details

### Communication Protocol

All messages are JSON-RPC 2.0 formatted, newline-delimited (`\n`):

**Request (Alice → Bob):**

```json
{ "jsonrpc": "2.0", "id": 1, "method": "echo", "params": { "message": "Hello" } }
```

**Response (Bob → Alice):**

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "echoed": "Hello", "timestamp": 1706572800000 } }
```

**Notification (Bob → Alice, no response expected):**

```json
{ "jsonrpc": "2.0", "method": "progress", "params": { "message": "Processing...", "percent": 50 } }
```

### Bidirectional Communication

Unlike traditional HTTP, JSON-RPC over STDIO is inherently bidirectional:

- **Alice calls Bob's methods** - Request-response pattern for operations
- **Bob sends notifications to Alice** - One-way messages for logs, progress, events
- **Both can be client and server** - No strict client/server roles

### Streaming via Notifications

To mimic HTTP streaming (like Hono Streaming), Bob sends multiple notifications:

```
Alice: "processWithProgress" request →
                                         ← Bob: progress notification (25%)
                                         ← Bob: progress notification (50%)
                                         ← Bob: progress notification (75%)
                                         ← Bob: final response (100%)
```

This demonstrates how streaming progress updates can be sent during long-running operations without breaking the request-response model.

## OpenRPC Specifications

This example uses [OpenRPC](https://spec.open-rpc.org/) to document the JSON-RPC APIs (similar to OpenAPI for REST):

- **[alice-openrpc.json](./alice-openrpc.json)** - Methods Alice exposes (progress, log)
- **[bob-openrpc.json](./bob-openrpc.json)** - Methods Bob exposes (health, echo, processWithProgress, shutdown)

OpenRPC provides:

- Machine-readable API documentation
- Type definitions for requests and responses
- Example messages
- Validation schemas

## Running the Example

### Prerequisites

- Go 1.23+ installed
- Deno installed

### Run Alice

```bash
cd json_rpc_example
go run alice/main.go
```

Alice will:

1. Spawn Bob as a child process
2. Establish JSON-RPC connection over stdio
3. Call Bob's `health` method
4. Call Bob's `echo` method
5. Call Bob's `processWithProgress` method (demonstrates streaming)
6. Receive progress notifications from Bob during processing
7. Call Bob's `shutdown` method
8. Wait for Bob to exit gracefully

### Expected Output

```
[Alice] Starting Alice (Go parent process)
[Alice] Started Bob (Deno child process)
[Bob stderr] [Bob] Starting JSON-RPC server over STDIO
[Bob stderr] [Bob] Waiting for requests from Alice...
[Alice] [info] Bob is ready to receive requests

[Alice] === Test 1: Health Check ===
[Bob stderr] [Bob] Received request: health
[Bob stderr] [Bob] Sent response for: health
[Alice] Health check passed: status=ok, timestamp=1706572800000

[Alice] === Test 2: Echo Message ===
[Bob stderr] [Bob] Received request: echo
[Bob stderr] [Bob] Sent response for: echo
[Alice] Echo result: Hello from Alice! (timestamp: 1706572800000)

[Alice] === Test 3: Process with Progress (Streaming) ===
[Bob stderr] [Bob] Received request: processWithProgress
[Alice] Progress: 20% - Processing item 1 of 5: apple
[Alice] Progress: 40% - Processing item 2 of 5: banana
[Alice] Progress: 60% - Processing item 3 of 5: cherry
[Alice] Progress: 80% - Processing item 4 of 5: date
[Alice] Progress: 100% - Processing item 5 of 5: elderberry
[Bob stderr] [Bob] Sent response for: processWithProgress
[Alice] Processing complete: processed=5 items in 1000ms
[Alice] Results: [APPLE BANANA CHERRY DATE ELDERBERRY]

[Alice] === Test 4: Graceful Shutdown ===
[Bob stderr] [Bob] Received request: shutdown
[Bob stderr] [Bob] Sent response for: shutdown
[Bob stderr] [Bob] Shutting down gracefully
[Alice] Shutdown response: Shutting down gracefully

[Alice] === All tests completed successfully ===
```

## Use Case for Terraform Provider

For the `terraform-provider-denobridge`, JSON-RPC over STDIO is ideal because:

1. **Security without complexity** - No need for mTLS, key exchange, or certificate management
2. **Cross-platform support** - Works on all Terraform-supported platforms
3. **Process lifecycle** - Child Deno process tied to provider lifecycle
4. **Bidirectional communication** - Provider can receive notifications from Deno scripts
5. **Structured protocol** - JSON-RPC provides clear contracts for CRUD operations

### Mapping Terraform Operations to JSON-RPC

| **Terraform Operation**  | **JSON-RPC Method**      | **Description**                     |
| ------------------------ | ------------------------ | ----------------------------------- |
| Resource Create          | `createResource`         | Create new resource, return state   |
| Resource Read            | `readResource`           | Read resource state                 |
| Resource Update          | `updateResource`         | Update resource, return new state   |
| Resource Delete          | `deleteResource`         | Delete resource                     |
| Data Source Read         | `readDataSource`         | Fetch external data                 |
| Ephemeral Resource Open  | `openEphemeralResource`  | Open short-lived resource           |
| Ephemeral Resource Close | `closeEphemeralResource` | Close short-lived resource          |
| Action                   | `executeAction`          | Execute operation                   |
| Progress Notification    | `progress`               | Stream progress updates to provider |
| Log Notification         | `log`                    | Send log messages to provider       |

### Benefits for Terraform Provider

1. **Type Safety** - OpenRPC specs can generate TypeScript types automatically
2. **Validation** - JSON-RPC libraries handle parameter validation
3. **Error Handling** - Standardized error codes and messages
4. **Debugging** - All messages are inspectable JSON
5. **Testing** - Easy to mock and test individual methods

## Comparison with Other Approaches

### vs. Unix Sockets

**Similarities:**

- Both rely on OS-level security
- Both are fast (no network overhead)
- Both have single-connection limitation

**Differences:**

- JSON-RPC/STDIO works on Windows; Unix sockets don't support named pipes well
- Unix sockets support multiple concurrent connections; STDIO doesn't
- Unix sockets can persist after process exit; STDIO is tied to process lifecycle

**Verdict:** JSON-RPC over STDIO is simpler and more portable. Use Unix sockets only if you need multiple concurrent connections.

### vs. HTTP with Custom ECDH Key Exchange

**Similarities:**

- Both work on all platforms
- Both provide secure communication

**Differences:**

- ECDH requires crypto libraries, key management, ratcheting
- ECDH has higher implementation complexity and maintenance burden
- ECDH requires careful security review to avoid vulnerabilities
- JSON-RPC/STDIO leverages OS security, no crypto needed

**Verdict:** JSON-RPC over STDIO is far simpler and equally secure for parent-child IPC. Custom crypto should be avoided unless absolutely necessary.

### vs. HTTP with mTLS

**Similarities:**

- Both work on all platforms
- Both provide cryptographic security

**Differences:**

- mTLS requires certificate generation and management
- mTLS has higher overhead (TLS handshake, encryption/decryption)
- mTLS depends on TLS library support (Deno has limitations)
- JSON-RPC/STDIO is simpler and faster

**Verdict:** JSON-RPC over STDIO is the better choice for local IPC. mTLS adds unnecessary complexity when OS provides sufficient security.

## Best Practices

### For Production Use in Terraform Provider

1. **Message Framing** ✅
   - Newline-delimited JSON is simple and works well
   - Consider length-prefixed framing for large messages
   - Buffer management to handle partial messages

2. **Error Handling** ✅
   - Use JSON-RPC error codes consistently
   - Map Deno errors to appropriate JSON-RPC error codes
   - Include detailed error messages for debugging

3. **Timeouts** ⚠️
   - Implement request timeouts in Alice
   - Long-running operations should send progress notifications
   - Consider context cancellation for graceful shutdown

4. **Process Supervision** ✅
   - Monitor child process health
   - Restart child on unexpected exit
   - Clean up resources on shutdown

5. **Logging** ✅
   - Bob sends logs via `log` notification
   - Alice forwards to Terraform logger
   - Consistent log levels

6. **Schema Validation** ⚠️
   - Use OpenRPC specs to generate validation code
   - Validate incoming requests and responses
   - Reject malformed messages early

## Recommendations

**For `terraform-provider-denobridge`, JSON-RPC over STDIO is the recommended approach:**

✅ **Pros:**

- Simple implementation (~200 lines per side)
- Cross-platform without platform-specific code
- OS-level security without crypto complexity
- Well-defined protocol (JSON-RPC 2.0)
- Low maintenance burden
- Easy to test and debug

❌ **Cons:**

- Single connection (not a problem for Terraform's use case)
- Not suitable for distributed systems (not needed)

**Verdict:** JSON-RPC over STDIO provides the best balance of simplicity, security, and maintainability for the Terraform provider's parent-child process communication needs.

## References

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [OpenRPC Specification](https://spec.open-rpc.org/)
- [sourcegraph/jsonrpc2 (Go)](https://pkg.go.dev/github.com/sourcegraph/jsonrpc2)
- [yieldray/json-rpc-ts (Deno)](https://jsr.io/@yieldray/json-rpc-ts)
- [Microsoft LSP over STDIO](https://microsoft.github.io/language-server-protocol/) - Production example of JSON-RPC/STDIO
