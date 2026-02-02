import { ActionProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

new ActionProvider<Props>({
  async invoke({ path, content }, progressCallback) {
    await progressCallback("about to write file");
    await Deno.writeTextFile(path, content);
    await progressCallback("file written");
  },
});
