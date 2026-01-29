import { Hono } from "jsr:@hono/hono";
import { crypto } from "jsr:@std/crypto";
import * as x509 from "npm:@peculiar/x509@1.12.3";

// Setup crypto provider for @peculiar/x509
x509.cryptoProvider.set(crypto);

interface CertMessage {
  cert: string;
}

interface BobResponse {
  port: number;
  cert: string;
}

// Read Alice's certificate from STDIN
const readStdin = async (): Promise<string> => {
  const decoder = new TextDecoder();
  const buf = new Uint8Array(4096);
  const n = await Deno.stdin.read(buf);
  if (n === null) {
    throw new Error("Failed to read from stdin");
  }
  return decoder.decode(buf.subarray(0, n)).trim();
};

// Generate self-signed certificate using @peculiar/x509
const generateSelfSignedCert = async (commonName: string) => {
  // Generate ECDSA key pair
  const keys = await crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true,
    ["sign", "verify"],
  );

  // Create certificate
  const cert = await x509.X509CertificateGenerator.createSelfSigned({
    name: `CN=${commonName}`,
    notBefore: new Date(),
    notAfter: new Date(Date.now() + 24 * 60 * 60 * 1000), // Valid for 24 hours
    signingAlgorithm: {
      name: "ECDSA",
      hash: "SHA-256",
    },
    keys,
    extensions: [
      new x509.KeyUsagesExtension(
        x509.KeyUsageFlags.digitalSignature | x509.KeyUsageFlags.keyEncipherment,
      ),
      new x509.ExtendedKeyUsageExtension([
        x509.ExtendedKeyUsage.serverAuth,
        x509.ExtendedKeyUsage.clientAuth,
      ]),
      new x509.SubjectAlternativeNameExtension([
        { type: "dns", value: "localhost" },
        { type: "ip", value: "127.0.0.1" },
        { type: "ip", value: "::1" },
      ]),
    ],
  });

  // Export certificate to PEM
  const certPEM = cert.toString("pem");

  // Export private key to PEM
  const privateKeyDER = await crypto.subtle.exportKey("pkcs8", keys.privateKey);
  const privateKeyPEM = formatPEM(new Uint8Array(privateKeyDER), "PRIVATE KEY");

  return {
    privateKeyPEM,
    certPEM,
  };
};

// Helper to format binary data as PEM
const formatPEM = (data: Uint8Array, label: string): string => {
  const base64 = btoa(String.fromCharCode(...data));
  const formatted = base64.match(/.{1,64}/g)?.join("\n") || base64;
  return `-----BEGIN ${label}-----\n${formatted}\n-----END ${label}-----\n`;
};

// Main execution
const main = async () => {
  // Generate Bob's self-signed certificate
  console.error("Bob: Generating self-signed TLS certificate...");
  const { privateKeyPEM, certPEM } = await generateSelfSignedCert("Bob");
  console.error(`Bob: Generated certificate:\n${certPEM}`);

  // Read Alice's certificate from STDIN
  const stdinData = await readStdin();
  const aliceMsg: CertMessage = JSON.parse(stdinData);
  console.error(`Bob: Received Alice's certificate via STDIN:\n${aliceMsg.cert}`);

  // Calculate SHA-256 hash of Alice's certificate
  const certBytes = new TextEncoder().encode(aliceMsg.cert);
  const hashBuffer = await crypto.subtle.digest("SHA-256", certBytes);
  const expectedClientCertHash = Array.from(new Uint8Array(hashBuffer))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
  console.error(`Bob: Expected certificate hash: ${expectedClientCertHash}`);

  // Create Hono app
  const app = new Hono();

  app.post("/message", async (c) => {
    // Validate client certificate hash from header
    // (Workaround for Deno not supporting mTLS client verification)
    const clientCertHashHeader = c.req.header("X-Client-Cert-Hash");

    if (!clientCertHashHeader) {
      console.error("Bob: Request missing X-Client-Cert-Hash header");
      return c.json({ error: "Client certificate hash required" }, 401);
    }

    if (clientCertHashHeader !== expectedClientCertHash) {
      console.error("Bob: Client certificate hash mismatch");
      console.error(`Bob: Expected: ${expectedClientCertHash}`);
      console.error(`Bob: Received: ${clientCertHashHeader}`);
      return c.json({ error: "Invalid client certificate" }, 401);
    }

    console.error("Bob: Client certificate validated successfully!");
    console.error("Bob: Sending secret message to Alice...");

    // Return secret message
    return c.json({
      message: "Hello Alice! This is Bob's secret message over TLS.",
    });
  });

  // Start HTTPS server with self-signed certificate
  // Use port 0 to get a random available port
  const server = Deno.serve({
    hostname: "127.0.0.1",
    port: 0,
    cert: certPEM,
    key: privateKeyPEM,
    onListen: ({ port }) => {
      console.error(`Bob: HTTPS server listening on https://127.0.0.1:${port}`);

      // Send Bob's certificate and port to Alice via STDOUT
      const bobResponse: BobResponse = {
        port: port,
        cert: certPEM,
      };
      console.log(JSON.stringify(bobResponse));
      console.error("Bob: Sent certificate and port to Alice via STDOUT");
      console.error("Bob: Waiting for Alice's request...");
    },
  }, app.fetch);

  // Wait for server to finish
  await server.finished;
};

main();
