import { ActionProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  destination: string;
}

new ActionProvider<Props>({
  async invoke({ destination }, progressCallback) {
    await progressCallback(`launching rocket to ${destination}`);
    await progressCallback(`3...`);
    await progressCallback(`2...`);
    await progressCallback(`1...`);
    await progressCallback(`blast off`);
  },
});
