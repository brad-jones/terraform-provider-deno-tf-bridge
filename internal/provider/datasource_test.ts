import { Hono } from "hono";
import { serveTLS } from "./mtls.ts";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/read", async (c) => {
  const body = await c.req.json();
  const { value } = body.props;

  // Hash the value with SHA256
  const encoder = new TextEncoder();
  const data = encoder.encode(value);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashedValue = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");

  return c.json({ hashedValue });
});

await serveTLS(app.fetch);
