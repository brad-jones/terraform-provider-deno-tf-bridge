# Key Exchange Example

This example demonstrates a secure communication protocol between a Go client (Alice) and a Deno TypeScript HTTP server (Bob) using ECDH key exchange, authenticated encryption, replay protection, and forward secrecy.

## Overview

The example shows how to establish secure HTTP communication without mTLS, which is useful since Deno doesn't currently support mTLS (see [denoland/deno#26825](https://github.com/denoland/deno/issues/26825)).

**Architecture:**

1. Alice (Go) spawns Bob (Deno) as a child process
2. Public keys are exchanged via stdin/stdout
3. Both parties derive a shared secret using ECDH
4. HTTP communication is encrypted with AES-GCM
5. Keys are ratcheted after each message for forward secrecy

## Security Properties

### ✅ Cryptographic Primitives

- **X25519 (ECDH)** - Modern elliptic curve Diffie-Hellman for key exchange
- **AES-256-GCM** - Authenticated encryption providing confidentiality and integrity
- **HKDF-SHA256** - Key derivation function for key ratcheting
- **CSPRNG** - Cryptographically secure random number generation (crypto/rand in Go, crypto.subtle in Deno)

### ✅ Security Features

1. **Authenticated Encryption**: AES-GCM provides both confidentiality and authenticity of messages
2. **Replay Protection**: Sequence numbers prevent replay attacks; both sides validate strict monotonic increase
3. **Forward Secrecy**: HKDF-based key ratcheting ensures compromise of current key doesn't expose past messages
4. **Context Binding**: Additional Authenticated Data (AAD) includes sequence number, direction, and endpoint
5. **Implicit Authentication**: ECDH shared secret derivation provides authentication without additional primitives

## Why No Additional Authentication Token Is Needed

### The Key Insight

The parent-child process relationship provides authenticated key exchange:

```
Alice spawns Bob
    ↓
OS gives Alice exclusive access to Bob's stdin/stdout
    ↓
Messages on stdio ARE authenticated by OS process isolation
    ↓
Public keys exchanged over stdio are trustworthy
    ↓
ECDH shared secret derivation proves identity
```

### Why ECDH Provides Implicit Authentication

```
Shared Secret = ECDH(my_private_key, their_public_key)

For Alice and Bob to compute the same shared secret:
- Alice has: Alice's private key + Bob's public key (from stdio)
- Bob has: Bob's private key + Alice's public key (from stdio)
- Result: Both compute same shared secret

For an attacker to compute the shared secret:
- Needs EITHER Alice's private key OR Bob's private key
- Private keys never leave their respective processes
- Cannot forge valid encrypted messages without a private key
```

**The ECDH key agreement itself provides authentication** - only parties with the legitimate private keys can derive the correct shared secret.

## Threat Model

### ✅ Protected Against

1. **Network eavesdropping** - All HTTP traffic is encrypted
2. **Message tampering** - AES-GCM authentication prevents modification
3. **Replay attacks** - Sequence numbers prevent message replay
4. **Port-scanning attackers** - Cannot decrypt or forge messages without private keys
5. **Forward compromise** - Past messages remain secure after key ratcheting

### ❌ NOT Protected Against (But These Imply Complete System Compromise)

1. **Local attacker with root/admin access** - Can inspect process memory and extract keys
2. **Debugger/ptrace attacks** - Can attach to process and read memory
3. **Compromised operating system** - Can intercept stdio and process memory

**If an attacker can compromise stdio or process memory, they already have complete control.**

## Implementation Details

### Message Format

Encrypted messages contain:

```
[8 bytes: sequence number][12 bytes: nonce][N bytes: AES-GCM ciphertext + tag]
```

Response format (JSON):

```json
{
  "ack": true,
  "ratchet_seq": 1,
  "encrypted_data": [/* Uint8Array */]
}
```

### AAD Format

Additional Authenticated Data binds messages to context:

```
seq:{sequence}:{direction}:{endpoint}
Example: seq:1:alice-to-bob:/message
```

### Key Ratcheting

After successful message delivery (confirmed by acknowledgment), both parties derive a new key:

```
new_key = HKDF-SHA256(
    current_key,
    salt="denobridge-key-ratchet",
    info="ratchet:{counter}"
)
```

This ensures keys remain synchronized even if messages are lost in transit.

## Recommended Improvements for Production

While the cryptographic design is sound, consider these enhancements for production use in the Terraform provider:

### 1. Error Handling

- Don't crash on replay attacks; log and return error response
- Implement graceful degradation
- Add comprehensive security event logging

### 2. Session Management

- Add explicit session ID
- Implement request-response correlation
- Add connection timeout to detect dead processes
- Implement keepalive for long-running sessions

### 3. Key Management ✅

**Implemented:**

- ✅ **Automatic rekeying after N messages**: System triggers full rekey after 10,000 messages
- ✅ **Time-based rekeying**: Rekey after 1 hour of operation
- ✅ **Sequence number overflow protection**: Rekey at 2^63 to prevent overflow
- ✅ **Key window for message loss tolerance**: Maintains sliding window of last 3 keys
  - Enables decryption of out-of-order messages
  - Handles network packet loss scenarios
  - Automatically prunes old keys to limit memory usage

**Implementation Details:**

```go
// Go constants
const (
    MaxMessagesBeforeRekey = 10000
    MaxTimeBeforeRekey = 1 * time.Hour
    SeqOverflowThreshold = uint64(1 << 63)
    KeyWindowSize = 3
)

// TypeScript constants
const MAX_MESSAGES_BEFORE_REKEY = 10000;
const MAX_TIME_BEFORE_REKEY_MS = 60 * 60 * 1000; // 1 hour
const SEQ_OVERFLOW_THRESHOLD = 2n ** 63n;
const KEY_WINDOW_SIZE = 3;
```

The key window allows graceful handling of:

- Network packet reordering
- Message loss requiring retransmission with old keys
- Asynchronous message processing

When decryption fails with the current key, the system automatically attempts decryption with recent keys in the window.

**Note on Production Rekeying:**

In the current example, when a rekey condition is detected (message count, time limit, or sequence overflow), the system logs the event but continues operation. In a production implementation for the Terraform provider, a full rekey would:

1. Generate new ECDH key pairs for both parties
2. Exchange new public keys over the existing encrypted channel
3. Derive fresh directional keys from the new shared secret
4. Reset sequence numbers to 1
5. Clear the key window and start fresh
6. Continue operation seamlessly without dropping connection

This ensures perfect forward secrecy over the lifetime of long-running sessions.

### 4. Graceful Shutdown

Implement shutdown protocol:

```
1. Send encrypted "shutdown" message
2. Wait for acknowledgment
3. Close connections
4. Clean up resources
```

## Example Flow

```
Alice (Go)                          Bob (Deno)
    |                                   |
    |-- spawn process ---------------->|
    |                                   |
    |-- Alice's public key (stdio) --->|
    |<-- Bob's public key + port ------|
    |                                   |
    | [Both derive shared secret]       |
    |                                   |
    |-- POST /message (encrypted) ---->|
    |                                   | [decrypt, validate seq]
    |                                   | [process message]
    |                                   | [ratchet key]
    |<-- JSON response (ack + enc) ----|
    |                                   |
    | [decrypt, validate seq]           |
    | [verify ack received]             |
    | [ratchet key on ack]              |
    |                                   |
```

## Running the Example

```bash
cd key_exchange_example
go run alice/main.go
```

Alice will:

1. Generate her key pair
2. Spawn Bob as a child process
3. Exchange public keys via stdio
4. Send an encrypted HTTP message to Bob
5. Receive and decrypt Bob's response
6. Ratchet keys for forward secrecy
7. Terminate Bob's process

## Note on Logging

This example exports keys and logs sensitive information for educational purposes. **Never do this in production:**

- Keys are marked as `extractable: true` in Deno to enable logging
- Shared secrets are logged in base64
- Messages are logged in plaintext after decryption

In production, keys should be non-extractable and sensitive data should never be logged.

## References

- [X25519](https://cr.yp.to/ecdh.html) - Elliptic Curve Diffie-Hellman
- [AES-GCM](https://csrc.nist.gov/publications/detail/sp/800-38d/final) - Authenticated Encryption
- [HKDF](https://tools.ietf.org/html/rfc5869) - HMAC-based Key Derivation Function
- [Signal Protocol](https://signal.org/docs/) - Double Ratchet algorithm for reference
- [Deno mTLS Issue](https://github.com/denoland/deno/issues/26825)
