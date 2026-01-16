import { Hono } from "jsr:@hono/hono";
import { streamText } from "jsr:@hono/hono/streaming";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/invoke", async (c) => {
  const body = await c.req.json();
  const { destination } = body.props;
  c.header("Content-Type", "application/jsonl");

  return streamText(c, async (stream) => {
    await stream.writeln(JSON.stringify({ message: `launching rocket to ${destination}` }));
    await stream.writeln(JSON.stringify({ message: "3..." }));
    await stream.writeln(JSON.stringify({ message: "2..." }));
    await stream.writeln(JSON.stringify({ message: "1..." }));
    await stream.writeln(JSON.stringify({ message: "blast off" }));
  });
});

export default app satisfies Deno.ServeDefaultExport;
