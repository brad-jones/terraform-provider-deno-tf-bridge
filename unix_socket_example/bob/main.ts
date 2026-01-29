import { Hono } from "hono";

const app = new Hono();

app.get("/ping", (c) => c.json({ pong: "ping" }));

await Deno.serve({ transport: "unix", path: `${import.meta.dirname}/http.sock` }, app.fetch).finished;
