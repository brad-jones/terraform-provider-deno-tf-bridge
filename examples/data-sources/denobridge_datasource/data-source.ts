import { Hono } from "jsr:@hono/hono";
import { serveTLS } from "../../mtls.ts";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/read", async (c) => {
  const body = await c.req.json();
  const { query, recordType } = body.props;
  const result = await Deno.resolveDns(query, recordType, {
    nameServer: { ipAddr: "1.1.1.1", port: 53 },
  });
  return c.json(result);
});

await serveTLS(app.fetch);
