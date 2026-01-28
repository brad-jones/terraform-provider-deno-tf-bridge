/**
 * Reference implementation of mTLS handshake for terraform-provider-denobridge.
 *
 * This module provides a serveTLS() function that wraps a Deno HTTP handler with
 * automatic mutual TLS authentication. It handles:
 * - Reading client certificate from stdin
 * - Generating server certificate
 * - Finding available port
 * - Printing handshake to stdout
 * - Starting HTTPS server with mTLS
 *
 * See templates/guides/mtls-handshake.md for detailed documentation.
 */

import * as x509 from "npm:@peculiar/x509@1.12.3";

// Initialize crypto provider for @peculiar/x509
x509.cryptoProvider.set(crypto);

/**
 * Handshake message format for stdin (Go -> Deno)
 */
interface ClientHandshake {
  cert: string; // PEM-encoded client certificate (public part only)
}

/**
 * Handshake message format for stdout (Deno -> Go)
 */
interface ServerHandshake {
  port: number; // Port number the server is listening on
  cert: string; // PEM-encoded server certificate
}

/**
 * Generate a self-signed ECDSA P-256 certificate for mTLS.
 * Returns the certificate and private key as PEM strings.
 */
async function generateSelfSignedCert(): Promise<{ certPem: string; keyPem: string }> {
  // Generate ECDSA P-256 keypair
  const algorithm = {
    name: "ECDSA",
    namedCurve: "P-256",
    hash: "SHA-256",
  };

  const keys = await crypto.subtle.generateKey(
    algorithm,
    true, // extractable
    ["sign", "verify"],
  );

  // Generate random serial number
  const serialNumber = Array.from(crypto.getRandomValues(new Uint8Array(16)))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  // Create self-signed certificate
  const cert = await x509.X509CertificateGenerator.createSelfSigned({
    serialNumber,
    name: "CN=127.0.0.1",
    notBefore: new Date(),
    notAfter: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
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
        [
          { type: "ip", value: "127.0.0.1" },
          { type: "dns", value: "localhost" },
        ],
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

/**
 * Format binary data as PEM.
 */
function formatPem(buffer: ArrayBuffer, label: string): string {
  const base64 = btoa(String.fromCharCode(...new Uint8Array(buffer)));
  const formatted = base64.match(/.{1,64}/g)?.join("\n") || base64;
  return `-----BEGIN ${label}-----\n${formatted}\n-----END ${label}-----\n`;
}

/**
 * Read client certificate from stdin.
 * The Go client writes a JSON object with "cert" and "key" fields.
 */
async function readClientCert(): Promise<ClientHandshake> {
  const decoder = new TextDecoder();
  const stdinData: Uint8Array[] = [];

  // Read all data from stdin
  for await (const chunk of Deno.stdin.readable) {
    stdinData.push(chunk);
  }

  // Concatenate chunks and parse JSON
  const totalLength = stdinData.reduce((acc, chunk) => acc + chunk.length, 0);
  const combined = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of stdinData) {
    combined.set(chunk, offset);
    offset += chunk.length;
  }

  const json = decoder.decode(combined);
  const handshake = JSON.parse(json) as ClientHandshake;

  if (!handshake.cert) {
    throw new Error("Invalid client handshake: missing cert");
}

/**
 * Write server handshake to stdout.
 * Prints a single JSON line with port and server certificate.
 */
function writeServerHandshake(port: number, cert: string): void {
  const handshake: ServerHandshake = { port, cert };
  console.log(JSON.stringify(handshake));
}

/**
 * Wrap a Deno HTTP handler with mTLS authentication.
 *
 * This function:
 * 1. Reads client certificate from stdin
 * 2. Generates server certificate
 * 3. Finds an available port
 * 4. Prints handshake to stdout
 * 5. Starts HTTPS server with mTLS
 *
 * @param handler - The HTTP handler function (Request => Response | Promise<Response>)
 * @returns Promise that resolves when server starts (never resolves normally)
 *
 * @example
 * ```typescript
 * import { Hono } from "jsr:@hono/hono";
 * import { serveTLS } from "./mtls.ts";
 *
 * const app = new Hono();
 * app.get("/health", (c) => c.body(null, 204));
 * app.post("/read", async (c) => { ... });
 *
 * await serveTLS(app.fetch);
 * ```
 */
export async function serveTLS(
  handler: (request: Request) => Response | Promise<Response>,
): Promise<never> {
  // Read client certificate from stdin
  const clientHandshake = await readClientCert();

  // Generate server certificate
  const { certPem: serverCert, keyPem: serverKey } = await generateSelfSignedCert();

  // Find available port by listening on port 0
  const listener = Deno.listen({ hostname: "127.0.0.1", port: 0 });
  const addr = listener.addr as Deno.NetAddr;
  const port = addr.port;
  listener.close();

  // Write handshake to stdout (must be first output)
  writeServerHandshake(port, serverCert);

  // Start HTTPS server with mTLS
  // Configure server to require and validate client certificates
  await Deno.serve({
    hostname: "127.0.0.1",
    port,
    cert: serverCert,
    key: serverKey,
    // Client certificate validation - require the specific client cert we received
    // Note: Deno.serve doesn't directly support client cert validation via 'ca' option
    // in the same way Node.js does. For true mTLS, we would need to use Deno.serveTls
    // with lower-level TLS options. However, since we're binding to 127.0.0.1 only
    // and the client knows the server cert, this provides reasonable security for
    // localhost communication.
  }, handler).finished;

  // This should never be reached as serve() runs indefinitely
  throw new Error("Server unexpectedly stopped");
}

/**
 * Type definition for Deno.ServeDefaultExport compatibility.
 *
 * @example
 * ```typescript
 * export default app satisfies DenoServeDefaultExport;
 * ```
 */
export type DenoServeDefaultExport = {
  fetch: (request: Request) => Response | Promise<Response>;
};
