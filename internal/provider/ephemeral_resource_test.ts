import { Hono } from "hono";
import { serveTLS } from "./mtls.ts";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/open", (c) => {
  return c.json({ result: { uuid: crypto.randomUUID() } });
});

await serveTLS(app.fetch);
