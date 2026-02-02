import { TextLineStream } from "@std/streams";
import { JSONRPCClient, type JSONRPCMethods, JSONRPCServer } from "@yieldray/json-rpc-ts";

/**
 * Represents a readable stream that can be explicitly closed.
 *
 * This interface wraps a ReadableStream with a close method, allowing for
 * graceful shutdown of the stream when it's no longer needed. It's commonly
 * used with stdin or other input sources that need explicit cleanup.
 *
 * @example
 * ```ts
 * const reader: CloseableReadable = {
 *   readable: Deno.stdin.readable,
 *   close: () => Deno.stdin.close()
 * };
 * ```
 */
export interface CloseableReadable {
  /**
   * The underlying readable stream of binary data.
   */
  readable: ReadableStream<Uint8Array<ArrayBuffer>>;

  /**
   * Closes the readable stream and releases any associated resources.
   */
  close(): void;
}

/**
 * Represents a writable stream for sending binary data.
 *
 * This interface wraps a WritableStream, typically used with stdout or other
 * output destinations for JSON-RPC communication. The stream accepts binary
 * data that will be written to the underlying output.
 *
 * @example
 * ```ts
 * const writer: Writeable = {
 *   writable: Deno.stdout.writable
 * };
 * ```
 */
export interface Writeable {
  /**
   * The underlying writable stream for binary data output.
   */
  writable: WritableStream<Uint8Array<ArrayBufferLike>>;
}

export interface JSocketOptions {
  /**
   * If enabled, verbose logs will be output on STDERR.
   */
  debugLogging: boolean;

  /**
   * How many seconds to wait for a response before timing out.
   *
   * Defaults to 30
   */
  responseTimeoutSec?: number;
}

/**
 * Creates a new JSocket with automatic type inference for ServerMethods.
 *
 * This is the recommended way to create a {@link JSocket} instance. Unlike calling the
 * constructor directly, this helper only requires you to specify the `ClientMethods` type
 * parameter - TypeScript will automatically infer the `ServerMethods` type from your
 * implementation, resulting in cleaner and more maintainable code.
 *
 * @template ClientMethods - The interface defining methods available on the remote party
 *
 * @param reader - The readable stream to receive JSON-RPC messages from (e.g., stdin)
 * @param writer - The writable stream to send JSON-RPC messages to (e.g., stdout)
 * @param options - Optional configuration for debug logging and timeout settings
 *
 * @returns A curried function that accepts the server methods implementation and returns a JSocket instance
 *
 * @see {@link JSocket} for detailed documentation on the returned instance
 *
 * @example
 * ```ts
 * type RemoteMethods = {
 *   add(params: { a: number; b: number }): Promise<number>;
 *   log(params: { message: string }): void;
 * };
 *
 * // ServerMethods type is automatically inferred!
 * await using socket = createJSocket<RemoteMethods>(
 *   Deno.stdin.readable,
 *   Deno.stdout.writable,
 *   { debugLogging: true, responseTimeoutSec: 60 }
 * )((client) => ({
 *   // This method can call back to the remote party
 *   async multiply(params: { x: number; y: number }) {
 *     client.log({ message: `Computing ${params.x} * ${params.y}` });
 *     return params.x * params.y;
 *   },
 *   async divide(params: { numerator: number; denominator: number }) {
 *     if (params.denominator === 0) {
 *       throw new Error("Division by zero");
 *     }
 *     return params.numerator / params.denominator;
 *   }
 * }));
 *
 * // Now you can call remote methods
 * const sum = await socket.client.add({ a: 5, b: 3 });
 * ```
 */
export function createJSocket<ClientMethods extends JSONRPCMethods>(
  reader: CloseableReadable,
  writer: Writeable,
  options?: JSocketOptions,
) {
  return <ServerMethods extends JSONRPCMethods>(
    serverMethods: (client: JSONRPCClient<ClientMethods>) => ServerMethods,
  ): JSocket<ClientMethods, ServerMethods> => {
    return new JSocket(reader, writer, serverMethods, options);
  };
}

/**
 * A bidirectional JSON-RPC 2.0 communication layer over stdio-like streams.
 *
 * JSocket provides both client and server capabilities in a single instance, enabling
 * full-duplex JSON-RPC communication. It handles message serialization, routing,
 * request/response matching, timeouts, and graceful cleanup. Messages are exchanged
 * as newline-delimited JSON over the provided streams.
 *
 * Key features:
 * - Bidirectional communication: Make calls in both directions simultaneously
 * - Type-safe: Full TypeScript support with generic method signatures
 * - Stream-based: Works with any ReadableStream/WritableStream (stdin/stdout, sockets, etc.)
 * - Timeout handling: Configurable timeouts for requests
 * - Async disposal: Clean shutdown with `await using` or manual disposal
 * - Debug logging: Optional verbose logging to STDERR
 *
 * @template ClientMethods - The interface defining methods available on the remote party
 * @template ServerMethods - The interface defining methods this instance exposes
 *
 * @remarks
 * **Recommended:** Use the {@link createJSocket} helper function instead of calling this
 * constructor directly. The helper provides better type inference - you only need to specify
 * the `ClientMethods` type parameter, and TypeScript will automatically infer `ServerMethods`
 * from your server methods implementation. This results in cleaner, more maintainable code.
 *
 * @example
 * ```ts
 * // Using createJSocket (recommended - automatic ServerMethods inference)
 * type RemoteMethods = {
 *   add(params: { a: number; b: number }): Promise<number>;
 * };
 *
 * await using socket = createJSocket<RemoteMethods>(
 *   Deno.stdin.readable,
 *   Deno.stdout.writable,
 *   { debugLogging: true }
 * )((client) => ({
 *   async multiply(params: { x: number; y: number }) {
 *     const sum = await client.add({ a: params.x, b: params.y });
 *     return params.x * params.y;
 *   }
 * }));
 *
 * // Using constructor directly (requires both type parameters)
 * type LocalMethods = { multiply(params: { x: number; y: number }): Promise<number> };
 * await using socket = new JSocket<RemoteMethods, LocalMethods>(
 *   Deno.stdin.readable,
 *   Deno.stdout.writable,
 *   (client) => ({
 *     async multiply(params: { x: number; y: number }) {
 *       return params.x * params.y;
 *     }
 *   })
 * );
 * ```
 */
export class JSocket<ClientMethods extends JSONRPCMethods, ServerMethods extends JSONRPCMethods>
  implements AsyncDisposable {
  /**
   * Use me to send requests and notifications to the remote party.
   */
  readonly client: JSONRPCClient<ClientMethods>;

  /**
   * This listens to requests and notifications from the remote party & routes them to your methods.
   */
  readonly server: JSONRPCServer<ServerMethods>;

  /**
   * Promise that resolves when the stream listener completes processing all messages.
   * Used during disposal to ensure cleanup waits for all handlers to finish.
   *
   * @internal
   */
  readonly #listener;

  /**
   * The reader used to receive JSON-RPC messages from the remote party.
   *
   * @internal
   */
  readonly #reader!: CloseableReadable;

  /**
   * The writable stream used to send JSON-RPC messages to the remote party.
   *
   * @internal
   */
  readonly #writer!: WritableStream;

  /**
   * Registry of custom listeners attached via the Rx method.
   * Each listener is called for every received line of input.
   *
   * @internal
   */
  readonly #listeners = new Map<(line: string) => Promise<void> | void, boolean>();

  /**
   * Tracks pending client requests awaiting responses, keyed by JSON-RPC message ID.
   * Each entry contains a resolver function and timeout handle for the request.
   *
   * @internal
   */
  readonly #clientResponseWaiters = new Map<number, { resolve: (response: string) => void; timeout: number }>();

  #running = false;

  /**
   * Creates a new bidirectional JSON-RPC connection over stdio-like streams.
   *
   * This constructor initializes both the client and server components, sets up message
   * routing, and starts listening for incoming messages on the reader stream. The client
   * is passed to the server methods factory function to enable bidirectional communication,
   * allowing server methods to make calls or send notifications back to the remote party.
   *
   * @param reader - The readable stream to receive JSON-RPC messages from (e.g., stdin)
   * @param writer - The writable stream to send JSON-RPC messages to (e.g., stdout)
   * @param serverMethods - Factory function that receives the client and returns an object
   *                        containing the methods that the server will expose
   * @param options - Optional configuration for debug logging and timeout settings
   *
   * @example
   * ```ts
   * const socket = new JSocket(
   *   Deno.stdin.readable,
   *   Deno.stdout.writable,
   *   (client) => ({
   *     async greet(params: { name: string }) {
   *       return { message: `Hello, ${params.name}!` };
   *     }
   *   }),
   *   { debugLogging: true }
   * );
   * ```
   */
  constructor(
    reader: CloseableReadable,
    writer: Writeable,
    serverMethods: (client: JSONRPCClient<ClientMethods>) => ServerMethods,
    readonly options?: JSocketOptions,
  ) {
    // Save the reader & writer for future use
    this.#writer = writer.writable;
    this.#reader = reader;

    // Build the client and server instances
    // We pass the client into the server to enable server methods to make calls to the other side.
    // eg: Notifications about progress or other bi-directional use cases.
    this.client = new JSONRPCClient<ClientMethods>(this.#clientHandler.bind(this));
    this.server = new JSONRPCServer<ServerMethods>(serverMethods(this.client));

    // Register the primary message processor
    this.Rx(this.#serverHandler.bind(this));

    // Start listening for lines on our reader
    this.#listener = (async () => {
      const jobs = new Map<Promise<void>, boolean>();

      this.#running = true;

      const reader = this.#reader.readable
        .pipeThrough(new TextDecoderStream())
        .pipeThrough(new TextLineStream())
        .getReader();

      try {
        while (this.#running) {
          const { value, done } = await reader.read();
          if (done) break;

          if (value) {
            this.#log(`Rx: ${value}`);

            // Pass the line to all registered listeners
            for (const [listener] of this.#listeners) {
              // Normalize the return value of the listener into a promise
              const job = (async () => {
                await listener(value);
              })();

              // Add the job to our jobs map
              jobs.set(job, true);

              // Ensure we remove the job once it's finished
              job.then((_) => jobs.delete(job));
            }
          }
        }
      } finally {
        reader.releaseLock();
      }

      // Wait for any remaining jobs to finish
      await Promise.all(jobs.keys());
    })();
  }

  /**
   * Low-level receive, provide a function that will be called upon each received line.
   *
   * Many listeners can be added. To stop listening, a cancellation function is returned
   * which can be called as needed. Alternatively supply an `AbortSignal` for cancellation.
   *
   * ```ts
   * const cancel = conn.Rx(line => {
   *   if (line === "special value") {
   *     cancel();
   *   }
   * });
   *
   * const controller = new AbortController();
   * setTimeout(controller.abort, 123);
   * conn.Rx(line => {}, controller.signal);
   * ```
   */
  Rx(listener: (line: string) => Promise<void> | void, signal?: AbortSignal) {
    this.#listeners.set(listener, true);

    const cancel = () => {
      this.#listeners.delete(listener);
    };

    if (signal) {
      signal.addEventListener("abort", cancel);
    }

    return cancel;
  }

  /**
   * Low-level transmit, call me to put a message on the wire.
   *
   * ```ts
   * await conn.Tx(`{ "foo": "bar" }`);
   * ```
   */
  async Tx(line: string) {
    let r;
    const w = this.#writer.getWriter();
    await w.ready;
    try {
      r = await w.write(new TextEncoder().encode(`${line}\n`));
      this.#log(`Tx: ${line}`);
    } finally {
      w.releaseLock();
    }
    return r;
  }

  /**
   * Logs the message to STDERR if debugLogging is enabled.
   * STDERR is normally used for logging when performing JSON-RPC over STDIO.
   *
   * @internal
   */
  #log(msg: string) {
    if (this.options?.debugLogging) {
      console.error(msg);
    }
  }

  /**
   * The method we give to JSONRPCClient to put a new line on to the wire.
   *
   * @internal
   */
  async #clientHandler(line: string): Promise<string> {
    // Write the request/notification
    await this.Tx(line);

    // Parse to check if it's a request (has id) or notification (no id)
    const message = JSON.parse(line);
    if (message?.id === undefined) {
      // Notification - no response expected
      return "";
    }

    // Request - wait for response with timeout
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.#clientResponseWaiters.delete(message.id);
        reject(new Error("Request timeout waiting for response"));
      }, 1000 * (this.options?.responseTimeoutSec ?? 30));

      this.#clientResponseWaiters.set(message.id, { resolve, timeout });
    });
  }

  /**
   * The method we use to call JSONRPCServer.handleRequest amongst other duties.
   * This is registered as the first listener with the Rx method upon construction.
   *
   * @internal
   */
  async #serverHandler(line: string) {
    if (line.trim() === "") return;

    try {
      const message = JSON.parse(line);

      // Check if it's a response to one of our requests
      if (
        message.jsonrpc === "2.0" &&
        message.id !== undefined &&
        (message.result !== undefined || message.error !== undefined)
      ) {
        const waiter = this.#clientResponseWaiters.get(message.id);
        if (waiter) {
          clearTimeout(waiter.timeout);
          waiter.resolve(line);
          this.#clientResponseWaiters.delete(message.id);
        } else {
          this.#log(`WARN: response (#${message.id}) received without a matching waiter`);
        }
        return;
      }

      // Let JSONRPCServer route the request
      const response = await this.server.handleRequest(line);

      // Send any response back
      // Notifications won't have a response
      if (response && response.length > 0) {
        await this.Tx(response);
      }
    } catch (error) {
      // Send error response
      this.#log(`Error processing request: ${error}`);
      this.#log(`Error stack: ${error instanceof Error ? error.stack : "no stack"}`);
      await this.Tx(JSON.stringify({
        jsonrpc: "2.0",
        id: null,
        error: {
          code: -32700,
          message: "Parse error",
          data: String(error),
        },
      }));
    }
  }

  /**
   * Cleans up resources and gracefully shuts down the JSON-RPC connection.
   *
   * This method cancels the stream listener, stops processing incoming messages,
   * and waits for all pending message handlers to complete before resolving.
   * It's automatically called when using `await using` with this instance.
   *
   * @example
   * ```ts
   * // Automatic disposal with await using
   * await using socket = new JSocket(...);
   * // socket is automatically disposed at end of scope
   *
   * // Manual disposal
   * const socket = new JSocket(...);
   * await socket[Symbol.asyncDispose]();
   * ```
   */
  async [Symbol.asyncDispose](): Promise<void> {
    this.#running = false;
    this.#reader.close();
    await this.#listener;
  }
}
