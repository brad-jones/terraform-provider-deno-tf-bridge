import { EphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  type: "v4";
}

interface Result {
  uuid: string;
}

new EphemeralResourceProvider<Props, Result>({
  // deno-lint-ignore require-await
  async open({ type }) {
    if (type !== "v4") {
      throw new Error("Unsupported UUID type");
    }
    return { result: { uuid: crypto.randomUUID() } };
  },
});
