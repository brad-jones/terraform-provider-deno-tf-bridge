import { Hono } from "hono";
import { streamText } from "hono/streaming";
import { serveTLS } from "./mtls.ts";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/invoke", async (c) => {
  const body = await c.req.json();
  const { path, content } = body.props;
  c.header("Content-Type", "application/jsonl");

  return streamText(c, async (stream) => {
    await stream.writeln(JSON.stringify({ message: "about to write file" }));
    await Deno.writeTextFile(path, content);
    await stream.writeln(JSON.stringify({ message: "file written" }));
  });
});

await serveTLS(app.fetch);
