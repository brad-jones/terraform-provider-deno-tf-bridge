# Almost mTLS Example

This example demonstrates a secure TLS-based communication protocol between a Go client (Alice) and a Deno TypeScript HTTPS server (Bob), working around Deno's current lack of mTLS client certificate verification support.

## Overview

This approach provides TLS encryption and certificate-based authentication while working within Deno's limitations.

**Architecture:**

1. Alice (Go) generates a self-signed TLS certificate
2. Bob (Deno) generates a self-signed TLS certificate
3. Certificates are exchanged via stdin/stdout
4. Bob starts an HTTPS server using his certificate
5. Alice connects with TLS, trusting Bob's certificate
6. Alice sends her certificate in an HTTP header (workaround for Deno issue #26825)
7. Bob validates the certificate from the header matches what he received via stdin

## Why "Almost" mTLS?

**True mTLS** (mutual TLS) involves both parties authenticating each other during the TLS handshake:

- Server presents certificate to client (✅ Bob does this)
- Client presents certificate to server (❌ Deno can't verify this yet)

**This implementation** achieves the same security goals:

- ✅ TLS encryption for all traffic
- ✅ Server authentication (Alice validates Bob's cert)
- ✅ Client authentication (Bob validates Alice's cert from HTTP header)
- ✅ Future-ready for when Deno adds mTLS support (see [denoland/deno#26825](https://github.com/denoland/deno/issues/26825))

## Security Properties

### ✅ What This Provides

1. **TLS Encryption**: All HTTP traffic is encrypted using standard TLS
2. **Server Authentication**: Alice validates Bob's certificate
3. **Client Authentication**: Bob validates Alice's certificate (via header)
4. **Localhost Binding**: Server only listens on 127.0.0.1
5. **Certificate Validation**: Bob ensures the certificate in the header matches what he received via stdin

### Compared to Custom Crypto (key_exchange_example)

| Feature            | Almost mTLS              | Custom Crypto              |
| ------------------ | ------------------------ | -------------------------- |
| TLS Encryption     | ✅ Standard TLS          | ❌ Custom AES-GCM          |
| Key Exchange       | ✅ TLS handshake         | ❌ Custom ECDH             |
| Forward Secrecy    | ✅ TLS provides this     | ✅ Manual ratcheting       |
| Replay Protection  | ✅ TLS provides this     | ✅ Manual sequence numbers |
| Lines of Code      | ~200                     | ~500+                      |
| Crypto Maintenance | ✅ OS/Runtime handles it | ❌ Manual implementation   |
| Industry Standard  | ✅ Standard TLS          | ❌ Custom protocol         |

## Limitations

### Current Implementation

1. **Simplified Certificate Generation**: Bob's certificate generation is simplified for demonstration. In production, use a proper X.509 certificate library for Deno.

2. **Certificate in Header**: The client certificate is sent in an HTTP header instead of during TLS handshake. This is functionally equivalent for authentication but not true mTLS.

3. **No Certificate Revocation**: No CRL or OCSP checking (appropriate for ephemeral self-signed certs).

### When Deno Adds mTLS Support

Once [denoland/deno#26825](https://github.com/denoland/deno/issues/26825) is resolved, you can update Bob to:

```typescript
Deno.serve({
  hostname: "127.0.0.1",
  port: 0,
  cert: certPEM,
  key: privateKeyPEM,
  // Future: Enable client certificate verification
  requestCert: true,
  rejectUnauthorized: true,
  ca: [aliceMsg.cert], // Trust Alice's cert
}, app.fetch);
```

And remove the header-based validation, making this true mTLS.

## Implementation Details

### Certificate Exchange Flow

```
Alice generates cert
    ↓
Alice spawns Bob
    ↓
Alice → STDIN → Bob: Alice's certificate
    ↓
Bob generates cert
    ↓
Bob ← STDOUT ← Alice: Bob's cert + port number
    ↓
Both have each other's certificates
```

### TLS Connection Flow

```
Alice → Bob: HTTPS request with TLS
    ↓
TLS Handshake:
  - Bob presents his certificate
  - Alice validates Bob's certificate
  - Encrypted channel established
    ↓
Alice includes her certificate in X-Client-Cert header
    ↓
Bob validates header cert matches stdin cert
    ↓
Bob responds with encrypted message
```

### Advantages Over Bearer Token

While a bearer token (random string) would be simpler, TLS certificates provide:

1. **Standard cryptographic primitives**: X.509 certificates are industry standard
2. **Future compatibility**: Easy migration to true mTLS when Deno supports it
3. **Proper TLS**: Full TLS security properties (encryption, forward secrecy, etc.)
4. **Client certificate in handshake**: Alice sends her cert during TLS handshake (even though Bob can't verify it yet)

## Running the Example

```bash
cd almost_mtls_example
go run alice/main.go
```

Alice will:

1. Generate her TLS certificate
2. Spawn Bob as a child process
3. Exchange certificates via stdio
4. Connect to Bob's HTTPS server
5. Send a request with her certificate in the header
6. Receive Bob's encrypted response
7. Terminate Bob's process

## Production Considerations

For the Terraform provider:

1. **Certificate Generation**: Use a proper X.509 library in Deno (e.g., from npm)
2. **Certificate Lifecycle**: Generate fresh certs for each session
3. **Error Handling**: Graceful handling of TLS errors
4. **Logging**: Don't log certificates in production
5. **Timeouts**: Add appropriate connection timeouts
6. **Certificate Validation**: Consider adding expiration checking

## Migration Path

When Deno adds mTLS support:

1. Enable `requestCert: true` in `Deno.serve()`
2. Remove `X-Client-Cert` header validation
3. Trust Alice's certificate via `ca` option
4. Bob will automatically validate Alice's cert during TLS handshake

## Comparison to Other Approaches

| Approach      | Complexity | Security | Cross-Platform    | Deno Support |
| ------------- | ---------- | -------- | ----------------- | ------------ |
| Unix Sockets  | Low        | Medium   | ❌ Not on Windows | ❌           |
| Bearer Token  | Low        | Medium   | ✅                | ✅           |
| Almost mTLS   | Medium     | High     | ✅                | ✅           |
| Custom Crypto | High       | High     | ✅                | ✅           |
| True mTLS     | Medium     | High     | ✅                | ❌ Not yet   |

## References

- [Deno TLS Support](https://deno.land/manual/runtime/http_server_apis#tls-support)
- [Deno mTLS Issue #26825](https://github.com/denoland/deno/issues/26825)
- [TLS 1.3 Specification](https://datatracker.ietf.org/doc/html/rfc8446)
- [X.509 Certificates](https://datatracker.ietf.org/doc/html/rfc5280)
