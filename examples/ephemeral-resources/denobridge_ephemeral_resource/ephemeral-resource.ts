import { Hono } from "jsr:@hono/hono";
import { serveTLS } from "../../mtls.ts";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/open", async (c) => {
  const body = await c.req.json();
  const { type } = body.props;

  if (type !== "v4") {
    return c.json({ error: "Unsupported UUID type" }, 422);
  }

  return c.json(crypto.randomUUID());
});

await serveTLS(app.fetch);
