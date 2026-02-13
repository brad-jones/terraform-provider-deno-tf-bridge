import { ZodEphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";
import { z } from "@zod/zod";

const propsSchema = z.object({
  type: z.enum(["v4"]),
});

const resultSchema = z.object({
  uuid: z.uuid(),
  sensitive: z.object({
    secret: z.string(),
  }),
});

const privateDataSchema = z.never();

new ZodEphemeralResourceProvider(propsSchema, resultSchema, privateDataSchema, {
  open({ type }) {
    if (type !== "v4") throw new Error(`unsupported uuid type`);
    return Promise.resolve({
      result: {
        uuid: crypto.randomUUID(),
        sensitive: {
          secret: "ephemeral-secret",
        },
      },
    });
  },
});
