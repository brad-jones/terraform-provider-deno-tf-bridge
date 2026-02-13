// deno-lint-ignore-file require-await no-unused-vars

import { ResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

new ResourceProvider<Props>({
  async create({ path, content }) {
    await Deno.writeTextFile(path, content);
    return { id: path };
  },
  async read(id, props) {
    try {
      const content = await Deno.readTextFile(id);
      return {
        props: { path: id, content },
      };
    } catch (e) {
      if (e instanceof Deno.errors.NotFound) {
        return { exists: false };
      }
      throw e;
    }
  },
  async update(id, nextProps, currentProps) {
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }
    await Deno.writeTextFile(id, nextProps.content);
  },
  async delete(id, props) {
    await Deno.remove(id);
  },
  async modifyPlan(_id, planType, nextProps, currentProps) {
    if (planType !== "update") return;
    return { requiresReplacement: currentProps?.path !== nextProps?.path };
  },
});
