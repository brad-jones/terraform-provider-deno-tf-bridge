#!/usr/bin/env -S deno run -qA --ext=ts
import { Command } from "@cliffy/command";
import { $ } from "@david/dax";

await new Command()
  .name("try-cog-bump")
  .action(async () => {
    const dryRunResult = await $`cog bump -d --skip-untracked --auto`
      .captureCombined().noThrow();

    if (dryRunResult.code > 0) {
      if (dryRunResult.combined.includes("cause: No conventional commit found to bump current version.")) {
        console.log("nothing to do, no conventional commit found to bump current version.");
        Deno.exit(0);
      }

      console.error(dryRunResult.combined);
      Deno.exit(dryRunResult.code);
    }

    const result = await $`cog bump --auto`.noThrow();
    Deno.exit(result.code);
  })
  .parse();
