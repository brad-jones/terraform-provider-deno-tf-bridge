---
page_title: "mTLS Handshake Protocol"
description: |-
  Learn how the mTLS handshake works between the Terraform provider and Deno scripts
---

# mTLS Handshake Protocol

The terraform-provider-denobridge uses **mutual TLS (mTLS)** to secure communication between the Go provider process and the Deno HTTP server processes it spawns. This ensures that:

- Only the Go provider can connect to the Deno server
- Only the legitimate Deno server can be reached by the provider
- All communication is encrypted
- No other process on localhost can intercept or spoof requests

This guide explains the complete handshake protocol and how to implement it in your Deno scripts.

## Overview

The mTLS handshake follows this sequence:

1. **Go generates client certificate** - The provider creates an ephemeral ECDSA P-256 self-signed certificate
2. **Certificate sent via stdin** - Client cert and key are written as JSON to the Deno process's stdin
3. **Deno reads stdin** - Script reads the client certificate before doing anything else
4. **Deno generates server certificate** - Script creates its own ephemeral ECDSA P-256 certificate
5. **Deno finds port** - Script binds to an available port on 127.0.0.1
6. **Handshake sent via stdout** - Server port and certificate are printed as JSON (first output line)
7. **Go parses handshake** - Provider reads stdout, extracts server certificate and port
8. **HTTPS connection established** - Provider connects using HTTPS with mutual authentication
9. **Secure communication** - All subsequent HTTP requests use encrypted mTLS

## Protocol Details

### Stdin Protocol (Go → Deno)

The provider writes a single JSON object to the Deno process's stdin immediately after starting it:

**JSON Schema:**

```json
{
  "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"
}
```

**Fields:**

- `cert` (string): PEM-encoded X.509 client certificate (public part only)

**Security Note:** The client's private key is NOT sent - only the certificate (public portion) is shared. The private key remains on the client side and is used to establish the TLS connection. The server receives the client certificate to validate incoming connections.

**Important:** After writing this JSON, stdin is immediately closed (EOF). Your script must read all of stdin before attempting any other operations.

### Stdout Protocol (Deno → Go)

The Deno script must print a single JSON object as the **first line** of stdout:

**JSON Schema:**

```json
{
  "port": 12345,
  "cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"
}
```

**Fields:**

- `port` (number): TCP port number the HTTPS server is listening on (on 127.0.0.1)
- `cert` (string): PEM-encoded X.509 server certificate

**Important:** This must be the first output to stdout. The provider will wait up to 30 seconds for this handshake. Any other output before the handshake will cause the provider to fail.

### Certificate Requirements

Both client and server certificates must meet these requirements:

1. **Algorithm**: ECDSA with P-256 curve (secp256r1)
2. **Signature**: SHA-256
3. **Subject**: CN=127.0.0.1
4. **Subject Alternative Names**:
   - IP: 127.0.0.1
   - DNS: localhost
5. **Key Usage**: Digital Signature, Key Encipherment
6. **Extended Key Usage**: Server Authentication, Client Authentication
7. **Validity**: Any reasonable period (reference implementation uses 1 year)
8. **Self-signed**: Both certificates are self-signed

The provider will validate that the server certificate's hostname matches `127.0.0.1`.

## Reference Implementation

### Complete serveTLS() Function

The provider includes a reference implementation that handles the entire mTLS handshake. You can use this directly or adapt it to your needs.

**Location**: See `examples/mtls.ts` or `internal/provider/mtls.ts`

**Usage:**

```typescript
import { Hono } from "jsr:@hono/hono";
import { serveTLS } from "./mtls.ts";

const app = new Hono();

app.get("/health", (c) => c.body(null, 204));
app.post("/read", async (c) => {
  // Your resource logic here
  return c.json({/* ... */});
});

// Wrap your handler with serveTLS
await serveTLS(app.fetch);
```

### Step-by-Step Implementation

If you want to implement the handshake yourself, here's a complete walkthrough:

#### 1. Read Client Certificate from Stdin

```typescript
async function readClientCert(): Promise<{ cert: string }> {
  const decoder = new TextDecoder();
  const chunks: Uint8Array[] = [];

  // Read all data from stdin until EOF
  for await (const chunk of Deno.stdin.readable) {
    chunks.push(chunk);
  }

  // Concatenate and parse JSON
  const totalLength = chunks.reduce((acc, c) => acc + c.length, 0);
  const combined = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    combined.set(chunk, offset);
    offset += chunk.length;
  }

  const json = decoder.decode(combined);
  const handshake = JSON.parse(json);

  if (!handshake.cert) {
    throw new Error("Invalid client handshake: missing cert");
  }

  return handshake;
}
```

#### 2. Generate Server Certificate

You'll need to generate an ECDSA P-256 X.509 certificate. The reference implementation uses `npm:@peculiar/x509`:

```typescript
import * as x509 from "npm:@peculiar/x509@1.12.3";

// Initialize crypto provider
x509.cryptoProvider.set(crypto);

async function generateServerCert(): Promise<{ certPem: string; keyPem: string }> {
  // Generate ECDSA P-256 keypair
  const algorithm = {
    name: "ECDSA",
    namedCurve: "P-256",
    hash: "SHA-256",
  };

  const keys = await crypto.subtle.generateKey(algorithm, true, ["sign", "verify"]);

  // Generate random serial number
  const serialNumber = Array.from(crypto.getRandomValues(new Uint8Array(16)))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  // Create self-signed certificate
  const cert = await x509.X509CertificateGenerator.createSelfSigned({
    serialNumber,
    name: "CN=127.0.0.1",
    notBefore: new Date(),
    notAfter: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
    signingAlgorithm: algorithm,
    keys,
    extensions: [
      new x509.BasicConstraintsExtension(false, undefined, true),
      new x509.KeyUsagesExtension(
        x509.KeyUsageFlags.digitalSignature | x509.KeyUsageFlags.keyEncipherment,
        true,
      ),
      new x509.ExtendedKeyUsageExtension(
        [x509.ExtendedKeyUsage.serverAuth, x509.ExtendedKeyUsage.clientAuth],
        true,
      ),
      await x509.SubjectKeyIdentifierExtension.create(keys.publicKey),
      new x509.SubjectAlternativeNameExtension(
        [{ type: "ip", value: "127.0.0.1" }, { type: "dns", value: "localhost" }],
        false,
      ),
    ],
  });

  // Export certificate to PEM
  const certPem = cert.toString("pem");

  // Export private key to PEM
  const privateKeyData = await crypto.subtle.exportKey("pkcs8", keys.privateKey);
  const keyPem = formatPem(privateKeyData, "PRIVATE KEY");

  return { certPem, keyPem };
}

function formatPem(buffer: ArrayBuffer, label: string): string {
  const base64 = btoa(String.fromCharCode(...new Uint8Array(buffer)));
  const formatted = base64.match(/.{1,64}/g)?.join("\n") || base64;
  return `-----BEGIN ${label}-----\n${formatted}\n-----END ${label}-----\n`;
}
```

#### 3. Find Available Port

```typescript
function findAvailablePort(): number {
  // Listen on port 0 to let the OS assign an available port
  const listener = Deno.listen({ hostname: "127.0.0.1", port: 0 });
  const addr = listener.addr as Deno.NetAddr;
  const port = addr.port;
  listener.close(); // Close immediately to free the port
  return port;
}
```

#### 4. Print Handshake to Stdout

```typescript
function writeHandshake(port: number, cert: string): void {
  const handshake = { port, cert };
  console.log(JSON.stringify(handshake));
}
```

**Critical:** This must be the first line printed to stdout. Do not use `console.log()` for anything else before this.

#### 5. Start HTTPS Server

```typescript
async function startServer(
  port: number,
  serverCert: string,
  serverKey: string,
  handler: (req: Request) => Response | Promise<Response>,
): Promise<never> {
  await Deno.serve({
    hostname: "127.0.0.1",
    port,
    cert: serverCert,
    key: serverKey,
  }, handler).finished;

  throw new Error("Server stopped unexpectedly");
}
```

#### 6. Complete Main Function

```typescript
// Read client certificate from stdin
const clientHandshake = await readClientCert();

// Generate server certificate
const { certPem: serverCert, keyPem: serverKey } = await generateServerCert();

// Find available port
const port = findAvailablePort();

// Print handshake (MUST be first output!)
writeHandshake(port, serverCert);

// Start HTTPS server
await startServer(port, serverCert, serverKey, yourHandler);
```

## Security Considerations

### Why mTLS?

Even though the server binds to `127.0.0.1` (localhost only), mTLS provides important security benefits:

1. **Process Isolation**: Prevents other processes on the same machine from connecting to the Deno server
2. **Multi-user Systems**: On shared systems, other users cannot intercept communication
3. **Malware Protection**: Compromised local processes cannot tamper with Terraform operations
4. **Integrity**: Ensures no man-in-the-middle attacks, even on localhost
5. **Encryption**: All data is encrypted, including sensitive resource properties

### Certificate Lifecycle

- **Ephemeral**: Certificates are generated fresh for each Deno process
- **Short-lived**: Certificates are only valid while the process runs
- **Memory-only**: Private keys are never written to disk
- **Single-use**: Each certificate pair is used for only one provider-script connection

### Trust Model

The trust model is simple and secure:

1. The Go provider generates a client certificate
2. The Go provider passes the client certificate directly to the Deno process (its child)
3. The Deno process generates a server certificate
4. The Deno process passes the server certificate back via stdout
5. Both parties now trust each other's certificates explicitly
6. No certificate authority (CA) or external trust store is involved

This works because:

- The provider starts the Deno process (parent-child relationship)
- Communication happens over private stdin/stdout pipes
- No other process can intercept the certificate exchange
- Certificates are ephemeral and unique to this connection
- Only public certificates are exchanged - private keys never leave their respective processes

**Important Security Note:** The client only sends its certificate (public part), never its private key. The server can use this certificate to validate the client during the TLS handshake. Similarly, the server sends its certificate (public part) back to the client. Private keys remain private to each process and are used only to establish the encrypted connection.

## Troubleshooting

### "timeout waiting for handshake from Deno server"

**Cause**: The Deno script didn't print the handshake JSON within 30 seconds.

**Solutions**:

- Ensure `console.log(JSON.stringify({port, cert}))` is the first output
- Check that your script isn't printing debug information before the handshake
- Verify certificate generation completes successfully
- Check Deno stderr output for errors

### "failed to parse handshake JSON"

**Cause**: The first line of stdout wasn't valid JSON or missing required fields.

**Solutions**:

- Ensure the JSON includes both `port` (number) and `cert` (string) fields
- Check that no other output precedes the handshake
- Verify the certificate is properly PEM-encoded

### "failed to decode server certificate PEM"

**Cause**: The certificate in the handshake wasn't valid PEM format.

**Solutions**:

- Ensure certificate is exported as PEM (not DER)
- Verify the certificate includes the PEM header/footer lines
- Check that newlines are preserved in the PEM string

### "deno process exited during handshake"

**Cause**: The Deno script crashed before printing the handshake.

**Solutions**:

- Check Deno stderr output for error messages
- Verify `@peculiar/x509` or your certificate library is properly imported
- Ensure the script has permission to read stdin and bind to network ports
- Check that your Deno version supports the required APIs

## Alternative Implementations

While the reference implementation uses `@peculiar/x509`, you can use any approach that generates valid X.509 certificates:

- **@peculiar/x509**: Pure TypeScript, works in Deno (recommended)
- **Native crypto**: Web Crypto API for keypairs + manual X.509 DER encoding
- **OpenSSL wrapper**: Call OpenSSL via `Deno.Command` (requires OpenSSL installed)
- **Other JSR packages**: Any package that can generate self-signed X.509 certificates

The only requirements are:

1. Generate ECDSA P-256 certificate with CN=127.0.0.1
2. Export as PEM format
3. Include required extensions (SAN, Key Usage, Extended Key Usage)

## See Also

- [Deno Permissions Guide](./deno-permissions.md) - Understanding Deno's permission system
- [Resource Implementation](../resources/resource.md) - How to implement Terraform resources
- [Data Source Implementation](../data-sources/datasource.md) - How to implement data sources
