import { EphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Result {
  uuid: string;
}

new EphemeralResourceProvider<undefined, Result>({
  open(_props) {
    return Promise.resolve({ result: { uuid: crypto.randomUUID() } });
  },
});
