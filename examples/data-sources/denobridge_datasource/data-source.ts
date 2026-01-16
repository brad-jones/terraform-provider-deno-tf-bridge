import { Hono } from "jsr:@hono/hono";

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

export default app satisfies Deno.ServeDefaultExport;
