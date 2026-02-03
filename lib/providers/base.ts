import type { JSONRPCClient, JSONRPCMethods } from "@yieldray/json-rpc-ts";
import { createJSocket } from "../jsocket.ts";

export class BaseJsonRpcProvider<RemoteMethods extends JSONRPCMethods = JSONRPCMethods> {
  constructor(providerMethods: (client: JSONRPCClient<RemoteMethods>) => Record<string, unknown>) {
    console.error(
      "This is a JSON-RPC 2.0 server for the denobridge terraform provider. see: https://github.com/brad-jones/terraform-provider-denobridge",
    );

    let debugLogging = false;
    try {
      debugLogging = Deno.env.get("TF_LOG")?.toLowerCase() === "debug";
    } catch {
      // swallow exception due to no permissions to read env vars
    }

    const socket = createJSocket<RemoteMethods>(Deno.stdin, Deno.stdout, { debugLogging })(
      (client) => ({
        ...providerMethods(client),
        health() {
          return { ok: true };
        },
        shutdown() {
          console.error("Shutting down gracefully...");
          socket[Symbol.asyncDispose]();
        },
      }),
    );
  }
}
