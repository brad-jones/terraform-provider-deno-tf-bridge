import { JSONRPCClient, JSONRPCServer } from "jsr:@yieldray/json-rpc-ts";

// Types for JSON-RPC method parameters and results
interface HealthResult {
  status: "ok";
  timestamp: number;
}

interface EchoParams {
  message: string;
}

interface EchoResult {
  echoed: string;
  timestamp: number;
}

interface ProcessWithProgressParams {
  items: string[];
  delayMs?: number;
}

interface ProcessResult {
  processed: number;
  results: string[];
  duration: number;
}

interface ProgressParams {
  message: string;
  percent: number;
}

interface LogParams {
  level: "debug" | "info" | "warn" | "error";
  message: string;
}

interface ShutdownResult {
  message: string;
}

// Create RPC client for communicating with Alice
// The client needs a function that sends JSON-RPC messages to Alice and waits for responses
const encoder = new TextEncoder();
const aliceResponseWaiters = new Map<
  number,
  { resolve: (response: string) => void; timeout: number }
>();
let aliceRequestId = 1;

const aliceClient = new JSONRPCClient<{
  progress: (params: ProgressParams) => void;
  log: (params: LogParams) => void;
  getConfig: () => Promise<{
    maxRetries: number;
    timeout: number;
    enableDebug: boolean;
    environment: string;
  }>;
  shouldContinue: () => Promise<{ continue: boolean; reason: string }>;
}>(async (jsonString: string) => {
  // Write request/notification to stdout (Alice reads this)
  await Deno.stdout.write(encoder.encode(jsonString + "\n"));

  // Parse to check if it's a request (has id) or notification (no id)
  const message = JSON.parse(jsonString);
  if (message.id === undefined) {
    // Notification - no response expected
    return "";
  }

  // Request - wait for response from Alice with timeout
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      aliceResponseWaiters.delete(message.id);
      reject(new Error("Request timeout waiting for Alice response"));
    }, 30000); // 30 second timeout

    aliceResponseWaiters.set(message.id, { resolve, timeout });
  });
});

// Create RPC server for Bob's methods
const bobServer = new JSONRPCServer({
  health: async (): Promise<HealthResult> => {
    return {
      status: "ok",
      timestamp: Date.now(),
    };
  },

  echo: async (params: EchoParams): Promise<EchoResult> => {
    return {
      echoed: params.message,
      timestamp: Date.now(),
    };
  },

  processWithProgress: async (
    params: ProcessWithProgressParams,
  ): Promise<ProcessResult> => {
    const { items, delayMs = 100 } = params;
    const results: string[] = [];
    const startTime = Date.now();

    // Request configuration from Alice before processing
    const config = await aliceClient.request("getConfig", undefined);
    console.error(
      `[Bob] Received config from Alice: maxRetries=${config.maxRetries}, timeout=${config.timeout}`,
    );

    for (let i = 0; i < items.length; i++) {
      const item = items[i];

      // Send progress notification to Alice
      const percent = Math.round(((i + 1) / items.length) * 100);
      const progressMessage = `Processing item ${i + 1} of ${items.length}: ${item}`;

      // Send notification to Alice (no response expected)
      await aliceClient.notify("progress", {
        message: progressMessage,
        percent,
      });

      // Simulate work
      await new Promise((resolve) => setTimeout(resolve, delayMs));

      // Process the item (uppercase as example)
      results.push(item.toUpperCase());
    }

    const duration = Date.now() - startTime;

    return {
      processed: items.length,
      results,
      duration,
    };
  },

  shutdown: async (): Promise<ShutdownResult> => {
    // Schedule exit after response is sent
    setTimeout(() => {
      console.error("[Bob] Shutting down gracefully");
      Deno.exit(0);
    }, 100);

    return {
      message: "Shutting down gracefully",
    };
  },
});

// Handle incoming JSON-RPC requests from stdin
const decoder = new TextDecoder();
const buffer: string[] = [];

console.error("[Bob] Starting JSON-RPC server over STDIO");
console.error("[Bob] Waiting for requests from Alice...");

// Send initial ready notification to Alice
await aliceClient.notify("log", {
  level: "info",
  message: "Bob is ready to receive requests",
});

// Process incoming messages asynchronously to avoid blocking
async function processMessage(line: string) {
  if (line.trim() === "") return;

  try {
    const message = JSON.parse(line);

    // Check if it's a response to one of our requests to Alice
    if (
      message.jsonrpc === "2.0" && message.id !== undefined &&
      message.method === undefined
    ) {
      // This is a response from Alice
      const waiter = aliceResponseWaiters.get(message.id);
      if (waiter) {
        clearTimeout(waiter.timeout);
        waiter.resolve(line);
        aliceResponseWaiters.delete(message.id);
      }
      return;
    }

    // It's a request from Alice to Bob
    console.error(`[Bob] Received request: ${message.method}`);

    // Handle the request using bobServer and get JSON string response
    const responseStr = await bobServer.handleRequest(line);

    if (responseStr && responseStr.trim() !== "") {
      // Send response back to Alice via stdout
      await Deno.stdout.write(encoder.encode(responseStr + "\n"));
      console.error(`[Bob] Sent response for: ${message.method}`);
    } else {
      console.error(
        `[Bob] No response for notification/invalid request: ${message.method}`,
      );
    }
  } catch (error) {
    console.error(`[Bob] Error processing request: ${error}`);

    // Send error response
    const errorResponse = {
      jsonrpc: "2.0",
      id: null,
      error: {
        code: -32700,
        message: "Parse error",
        data: String(error),
      },
    };
    await Deno.stdout.write(
      encoder.encode(JSON.stringify(errorResponse) + "\n"),
    );
  }
}

// Read from stdin line by line
for await (const chunk of Deno.stdin.readable) {
  const text = decoder.decode(chunk);
  buffer.push(text);

  // Process complete lines
  const joined = buffer.join("");
  const lines = joined.split("\n");

  // Keep the last incomplete line in buffer
  buffer.length = 0;
  if (!joined.endsWith("\n")) {
    buffer.push(lines.pop()!);
  }

  // Process each complete line asynchronously (don't await)
  for (const line of lines) {
    processMessage(line); // Fire and forget - don't block the stdin loop
  }
}
