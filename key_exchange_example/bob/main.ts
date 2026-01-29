import { Hono } from "jsr:@hono/hono";
import { crypto } from "jsr:@std/crypto";
import { decodeBase64, encodeBase64 } from "jsr:@std/encoding";

const PROTOCOL_VERSION = "v1";

// Key management parameters
const MAX_MESSAGES_BEFORE_REKEY = 10000;
const MAX_TIME_BEFORE_REKEY_MS = 60 * 60 * 1000; // 1 hour
const SEQ_OVERFLOW_THRESHOLD = 2n ** 63n;
const KEY_WINDOW_SIZE = 3;

// KeyWindow maintains a sliding window of recent keys
class KeyWindow {
  private keys: CryptoKey[] = [];
  private ratchetCounters: bigint[] = [];

  add(key: CryptoKey, counter: bigint) {
    this.keys.push(key);
    this.ratchetCounters.push(counter);

    // Keep only the last KEY_WINDOW_SIZE keys
    if (this.keys.length > KEY_WINDOW_SIZE) {
      this.keys.shift();
      this.ratchetCounters.shift();
    }
  }

  get(counter: bigint): CryptoKey | null {
    const index = this.ratchetCounters.indexOf(counter);
    return index >= 0 ? this.keys[index] : null;
  }

  getAll(): { key: CryptoKey; counter: bigint }[] {
    return this.keys.map((key, i) => ({
      key,
      counter: this.ratchetCounters[i],
    }));
  }
}

interface PublicKeyMessage {
  public_key: string;
}

interface BobResponse {
  public_key: string;
  port: number;
}

interface MessageResponse {
  ack: boolean;
  ratchet_seq: number;
  encrypted_data: number[];
}

// Sequence tracking for replay protection
let sendSeq = 1n;
let recvSeq = 0n;
let ratchetCounter = 0n;
let messageCount = 0n;
let lastRekeyTime = Date.now();

// Key windows for handling out-of-order messages
const aliceToBobWindow = new KeyWindow();
const bobToAliceWindow = new KeyWindow();

// Read Alice's public key from STDIN
const readStdin = async (): Promise<string> => {
  const decoder = new TextDecoder();
  const buf = new Uint8Array(4096);
  const n = await Deno.stdin.read(buf);
  if (n === null) {
    throw new Error("Failed to read from stdin");
  }
  return decoder.decode(buf.subarray(0, n)).trim();
};

// Encrypt message using AES-GCM with shared secret
// Returns seqNum + nonce + ciphertext concatenated
const encryptMessage = async (
  sharedSecret: CryptoKey,
  plaintext: Uint8Array<ArrayBuffer>,
  seqNum: bigint,
  aad: Uint8Array<ArrayBuffer>,
): Promise<Uint8Array<ArrayBuffer>> => {
  // Generate random nonce
  const nonce = crypto.getRandomValues(new Uint8Array(12));

  // Encrypt with AAD (Additional Authenticated Data)
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: nonce, additionalData: aad },
    sharedSecret,
    plaintext,
  );

  // Prepend sequence number and nonce to ciphertext
  const seqBytes = new Uint8Array(8);
  const view = new DataView(seqBytes.buffer);
  view.setBigUint64(0, seqNum, false); // big-endian

  const result = new Uint8Array(
    8 + nonce.length + ciphertext.byteLength,
  );
  result.set(seqBytes, 0);
  result.set(nonce, 8);
  result.set(new Uint8Array(ciphertext), 8 + nonce.length);

  return result;
};

// Decrypt message encrypted with encryptMessage
// Expects seqNum + nonce + ciphertext concatenated
// Returns plaintext and sequence number
const decryptMessage = async (
  sharedSecret: CryptoKey,
  data: Uint8Array,
  aad: Uint8Array<ArrayBuffer>,
): Promise<{ plaintext: Uint8Array; seqNum: bigint }> => {
  if (data.length < 8 + 12) {
    throw new Error("Ciphertext too short");
  }

  // Extract sequence number
  const view = new DataView(data.buffer, data.byteOffset, 8);
  const seqNum = view.getBigUint64(0, false); // big-endian

  // Extract nonce and ciphertext
  const nonce = data.slice(8, 8 + 12);
  const ciphertext = data.slice(8 + 12);

  // Decrypt with AAD verification
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: nonce, additionalData: aad },
    sharedSecret,
    ciphertext,
  );

  return { plaintext: new Uint8Array(plaintext), seqNum };
};

// deriveDirectionalKey derives a direction-specific key from the shared secret
// This ensures Alice's sending key is different from Bob's sending key
const deriveDirectionalKey = async (
  sharedSecret: CryptoKey,
  direction: string,
): Promise<CryptoKey> => {
  const salt = new TextEncoder().encode("denobridge-directional-key");
  const info = new TextEncoder().encode(direction);

  // Export shared secret
  const sharedSecretBytes = await crypto.subtle.exportKey("raw", sharedSecret);

  // Derive directional key using HKDF
  const baseKey = await crypto.subtle.importKey(
    "raw",
    sharedSecretBytes,
    "HKDF",
    false,
    ["deriveKey"],
  );

  const directionalKey = await crypto.subtle.deriveKey(
    {
      name: "HKDF",
      hash: "SHA-256",
      salt: salt,
      info: info,
    },
    baseKey,
    {
      name: "AES-GCM",
      length: 256,
    },
    true,
    ["encrypt", "decrypt"],
  );

  return directionalKey;
};

// ratchetKey derives a new key from the current key using HKDF
// This provides forward secrecy - compromise of current key doesn't expose past messages
const ratchetKey = async (
  currentKey: CryptoKey,
  counter: bigint,
): Promise<CryptoKey> => {
  const salt = new TextEncoder().encode("denobridge-key-ratchet");
  const info = new TextEncoder().encode(`ratchet:${counter}`);

  // Export current key
  const currentKeyBytes = await crypto.subtle.exportKey("raw", currentKey);

  // Derive new key using HKDF
  const baseKey = await crypto.subtle.importKey(
    "raw",
    currentKeyBytes,
    "HKDF",
    false,
    ["deriveKey"],
  );

  const newKey = await crypto.subtle.deriveKey(
    {
      name: "HKDF",
      hash: "SHA-256",
      salt: salt,
      info: info,
    },
    baseKey,
    {
      name: "AES-GCM",
      length: 256,
    },
    true,
    ["encrypt", "decrypt"],
  );

  return newKey;
};

// Main execution
const main = async () => {
  // Read Alice's public key from STDIN
  const stdinData = await readStdin();
  const aliceMsg: PublicKeyMessage = JSON.parse(stdinData);
  console.error(`Bob: Received Alice's public key: ${aliceMsg.public_key}`);

  // Generate Bob's key pair
  const bobKeyPair = (await crypto.subtle.generateKey("X25519", true, [
    "deriveKey",
    "deriveBits",
  ])) as CryptoKeyPair;
  console.error("Bob: Generated key pair");

  // Export Bob's public key (raw format - 32 bytes)
  const bobPublicKeyBytes = await crypto.subtle.exportKey(
    "raw",
    bobKeyPair.publicKey,
  );
  const bobPublicKey = encodeBase64(bobPublicKeyBytes);

  // Import Alice's public key (raw format - 32 bytes)
  const alicePublicKeyBytes = decodeBase64(aliceMsg.public_key);
  const alicePublicKey = await crypto.subtle.importKey(
    "raw",
    alicePublicKeyBytes,
    "X25519",
    false,
    [],
  );

  // Derive shared secret
  const sharedSecret = await crypto.subtle.deriveKey(
    {
      name: "X25519",
      public: alicePublicKey,
    },
    bobKeyPair.privateKey,
    {
      name: "AES-GCM",
      length: 256,
    },
    true,
    ["encrypt", "decrypt"],
  );

  // Export shared secret for logging
  const sharedSecretBytes = await crypto.subtle.exportKey("raw", sharedSecret);
  console.error(
    `Bob: Derived shared secret: ${encodeBase64(sharedSecretBytes)}`,
  );

  // Derive separate keys for each direction
  let aliceToBobKey = await deriveDirectionalKey(sharedSecret, "alice-to-bob");
  let bobToAliceKey = await deriveDirectionalKey(sharedSecret, "bob-to-alice");
  const aliceToBobKeyBytes = await crypto.subtle.exportKey("raw", aliceToBobKey);
  const bobToAliceKeyBytes = await crypto.subtle.exportKey("raw", bobToAliceKey);
  console.error(
    `Bob: Derived alice-to-bob key: ${encodeBase64(aliceToBobKeyBytes)}`,
  );
  console.error(
    `Bob: Derived bob-to-alice key: ${encodeBase64(bobToAliceKeyBytes)}`,
  );

  // Store initial keys in windows
  aliceToBobWindow.add(aliceToBobKey, ratchetCounter);
  bobToAliceWindow.add(bobToAliceKey, ratchetCounter);

  // Create Hono app
  const app = new Hono();

  app.post("/message", async (c) => {
    // Read encrypted request body
    const encryptedRequest = new Uint8Array(await c.req.arrayBuffer());
    console.error(
      `Bob: Received encrypted message: ${encodeBase64(encryptedRequest)}`,
    );

    // Decrypt the message with replay protection and key window fallback
    const aadReq = new TextEncoder().encode(
      `${PROTOCOL_VERSION}:seq:${recvSeq + 1n}:alice-to-bob:/message`,
    );
    let decryptedMessage: Uint8Array | undefined;
    let requestSeq: bigint | undefined;

    try {
      const result = await decryptMessage(
        aliceToBobKey,
        encryptedRequest,
        aadReq,
      );
      decryptedMessage = result.plaintext;
      requestSeq = result.seqNum;
    } catch (err) {
      // Try previous keys in the window (handles message loss scenarios)
      console.error("Bob: Decryption failed with current key, trying key window...");
      let decrypted = false;
      const windowKeys = aliceToBobWindow.getAll();

      for (let i = windowKeys.length - 1; i >= 0; i--) {
        try {
          const result = await decryptMessage(
            windowKeys[i].key,
            encryptedRequest,
            aadReq,
          );
          decryptedMessage = result.plaintext;
          requestSeq = result.seqNum;
          console.error(
            `Bob: Successfully decrypted with key from window (ratchet_counter=${windowKeys[i].counter})`,
          );
          decrypted = true;
          break;
        } catch {
          // Continue trying other keys
        }
      }

      if (!decrypted) {
        console.error(`Bob: Failed to decrypt message with current key or key window: ${err}`);
        throw err;
      }
    }

    // Validate that we successfully decrypted a message
    if (decryptedMessage === undefined || requestSeq === undefined) {
      throw new Error("Failed to decrypt message");
    }

    // Validate sequence number to prevent replay attacks
    if (requestSeq <= recvSeq) {
      console.error(
        `Bob: REPLAY ATTACK DETECTED! Received seq=${requestSeq}, expected > ${recvSeq}`,
      );
      Deno.exit(1);
    }
    recvSeq = requestSeq;
    messageCount++;

    const messageText = new TextDecoder().decode(decryptedMessage);
    console.error(`Bob: Decrypted message (seq=${requestSeq}): ${messageText}`);

    // Prepare response
    const response = `Hello Alice! I received your message: "${messageText}"`;
    console.error(`Bob: Sending response: ${response}`);

    // Encrypt response with AAD
    const aadResp = new TextEncoder().encode(
      `${PROTOCOL_VERSION}:seq:${sendSeq}:bob-to-alice:/message`,
    );
    const encryptedResponse = await encryptMessage(
      bobToAliceKey,
      new TextEncoder().encode(response),
      sendSeq,
      aadResp,
    );
    console.error(
      `Bob: Encrypted response (seq=${sendSeq}): ${encodeBase64(encryptedResponse)}`,
    );
    sendSeq++;

    // Send acknowledgment - message was successfully received and processed
    const ack = true;

    // Ratchet both directional keys ONLY after successful message processing
    ratchetCounter++;
    // Store old keys in window before ratcheting
    aliceToBobWindow.add(aliceToBobKey, ratchetCounter - 1n);
    bobToAliceWindow.add(bobToAliceKey, ratchetCounter - 1n);

    aliceToBobKey = await ratchetKey(aliceToBobKey, ratchetCounter);
    bobToAliceKey = await ratchetKey(bobToAliceKey, ratchetCounter);
    const aliceToBobRatchetedBytes = await crypto.subtle.exportKey("raw", aliceToBobKey);
    const bobToAliceRatchetedBytes = await crypto.subtle.exportKey("raw", bobToAliceKey);
    console.error(
      `Bob: Ratcheted alice-to-bob key (counter=${ratchetCounter}): ${encodeBase64(aliceToBobRatchetedBytes)}`,
    );
    console.error(
      `Bob: Ratcheted bob-to-alice key (counter=${ratchetCounter}): ${encodeBase64(bobToAliceRatchetedBytes)}`,
    );

    // Check if we need to perform full rekey
    const timeSinceRekey = Date.now() - lastRekeyTime;
    let needRekey = false;
    let rekeyReason = "";

    if (sendSeq >= SEQ_OVERFLOW_THRESHOLD || recvSeq >= SEQ_OVERFLOW_THRESHOLD) {
      needRekey = true;
      rekeyReason = "sequence number overflow threshold reached";
    } else if (messageCount >= MAX_MESSAGES_BEFORE_REKEY) {
      needRekey = true;
      rekeyReason = `message count limit reached (${MAX_MESSAGES_BEFORE_REKEY})`;
    } else if (timeSinceRekey >= MAX_TIME_BEFORE_REKEY_MS) {
      needRekey = true;
      rekeyReason = `time limit reached (${MAX_TIME_BEFORE_REKEY_MS}ms)`;
    }

    if (needRekey) {
      console.error(`Bob: Initiating full rekey: ${rekeyReason}`);
      // In production, this would trigger a new key exchange
      // For this example, we'll just log it
      console.error("Bob: Rekey would reset sequence numbers and derive new keys from fresh ECDH exchange");
      console.error(
        `Bob: Current state - sendSeq=${sendSeq}, recvSeq=${recvSeq}, messageCount=${messageCount}, time=${timeSinceRekey}ms`,
      );
    }

    // Return JSON response with acknowledgment and encrypted data
    const messageResponse: MessageResponse = {
      ack: ack,
      ratchet_seq: Number(ratchetCounter),
      encrypted_data: Array.from(encryptedResponse),
    };

    return c.json(messageResponse);
  });

  // Start server on random port
  const server = Deno.serve({
    hostname: "127.0.0.1",
    port: 0, // Random available port
    onListen: ({ port }) => {
      console.error(`Bob: HTTP server listening on http://127.0.0.1:${port}`);

      // Send Bob's public key and port to Alice via STDOUT
      const bobResponse: BobResponse = {
        public_key: bobPublicKey,
        port: port,
      };
      console.log(JSON.stringify(bobResponse));
    },
  }, app.fetch);

  // Wait for server to finish
  await server.finished;
};

main();
