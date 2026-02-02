import { DatasourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  value: string;
}

interface Result {
  hashedValue: string;
}

new DatasourceProvider<Props, Result>({
  async read({ value }) {
    const encoder = new TextEncoder();
    const data = encoder.encode(value);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashedValue = hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
    return { hashedValue };
  },
});
