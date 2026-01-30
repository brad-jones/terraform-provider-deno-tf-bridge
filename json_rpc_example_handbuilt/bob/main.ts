import { createJSocket } from "./jsocket.ts";

type AliceMethods = {
  invokeProgress(params: { msg: string }): void;
};

const socket = createJSocket<AliceMethods>(
  Deno.stdin,
  Deno.stdout,
  { debugLogging: true },
)((client) => ({
  health() {
    return { ok: true };
  },
  echo(params: { message: string }) {
    return {
      echoed: params.message,
      timestamp: new Date().getTime(),
    };
  },
  async invoke(params: { count: number; delaySec: number }) {
    for (let i = 0; i < params.count; i++) {
      await new Promise((r) => setTimeout(r, 1000 * params.delaySec));
      await client.notify("invokeProgress", { msg: `index: ${i}...` });
    }
    return { itemsProcessed: params.count };
  },
  shutdown() {
    socket[Symbol.asyncDispose]();
  },
}));
