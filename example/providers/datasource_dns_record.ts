import { DatasourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  query: string;
  recordType: "A" | "AAAA" | "ANAME" | "CNAME" | "NS" | "PTR";
}

interface Result {
  ips: string[];
}

new DatasourceProvider<Props, Result>({
  async read({ query, recordType }) {
    return {
      ips: await Deno.resolveDns(query, recordType, {
        nameServer: { ipAddr: "1.1.1.1", port: 53 },
      }),
    };
  },
});
