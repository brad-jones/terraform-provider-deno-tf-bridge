import { ResourceProvider } from "@brad-jones/terraform-provider-denobridge";

interface Props {
  path: string;
  content: string;
}

interface State {
  mtime: number;
}

new ResourceProvider<Props, State>({
  async create({ path, content }) {
    await Deno.writeTextFile(path, content);
    return {
      id: path,
      state: {
        mtime: (await Deno.stat(path)).mtime!.getTime(),
      },
    };
  },
  async read(id, _props) {
    try {
      const content = await Deno.readTextFile(id);
      return {
        props: { path: id, content },
        state: {
          mtime: (await Deno.stat(id)).mtime!.getTime(),
        },
      };
    } catch (e) {
      if (e instanceof Deno.errors.NotFound) {
        return { exists: false };
      }
      throw e;
    }
  },
  async update(id, nextProps, currentProps, _currentState) {
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }
    await Deno.writeTextFile(id, nextProps.content);
    return { mtime: (await Deno.stat(id)).mtime!.getTime() };
  },
  async delete(id, _props, _state) {
    await Deno.remove(id);
  },
  async modifyPlan(_id, planType, nextProps, currentProps, _currentState) {
    if (planType !== "update") {
      return;
    }
    return { requiresReplacement: currentProps?.path !== nextProps.path };
  },
});
