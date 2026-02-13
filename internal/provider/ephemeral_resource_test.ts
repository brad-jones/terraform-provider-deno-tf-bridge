import { EphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  type: "v4";
}

interface Result {
  uuid: string;
  sensitive: {
    secret: string;
  };
}

new EphemeralResourceProvider<Props, Result>({
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
