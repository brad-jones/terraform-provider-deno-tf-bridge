import { EphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  type: "v4";
}

interface Result {
  uuid: string;
  sensitive: {
    token: string;
  };
}

new EphemeralResourceProvider<Props, Result>({
  // deno-lint-ignore require-await
  async open({ type }) {
    if (type !== "v4") {
      throw new Error("Unsupported UUID type");
    }
    return {
      result: {
        uuid: crypto.randomUUID(),
        // Values nested under the "sensitive" key are automatically
        // stored in the `sensitive_result` attribute in Terraform.
        sensitive: {
          token: "secret-api-token",
        },
      },
    };
  },
});
