#!/usr/bin/env -S deno run -qA --ext=ts
import { toText } from "@std/streams";
const input = await toText(Deno.stdin.readable);
const output = input.split("- - -")[1].trim() + "\n";
await Deno.stdout.write(new TextEncoder().encode(output));
