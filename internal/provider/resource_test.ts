// deno-lint-ignore-file require-await no-unused-vars

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
      // At minimum, the create endpoint must return an id that uniquely identifies this resource.
      id: path,

      // Optional additional state that is only known after the resource is created can be returned in the state object.
      state: {
        mtime: (await Deno.stat(path)).mtime!.getTime(),
      },
    };
  },
  async read(id, props) {
    // The read methods job is to get the current state of an existing resource.
    // And return an updated set of props & optional additional state that may
    // have changed (by outside actors), since the resource was initially created
    // (or last updated).
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
        // In the event we can no longer locate the resource we should signal
        // to tf to remove the resource from state. This will cause tf to re-create
        // the resource on the next plan.
        return { exists: false };
      }
      throw e;
    }
  },
  async update(id, nextProps, currentProps, currentState) {
    // In this case an in place update is not supported for a file when the path
    // changes because that would mean the ID would change too & tf can't deal with that easily.
    // So we return an error here but if an update wouldn't result in a new ID, there is no need to return such an error.
    // Or implement the /modify-plan endpoint, which should have caught this & told tf to do a replacement instead (ie: create then delete).
    if (nextProps.path !== currentProps.path) {
      throw new Error("Cannot change file path - requires resource replacement");
    }

    // Perform the update
    await Deno.writeTextFile(id, nextProps.content);

    // Return the updated state
    return { mtime: (await Deno.stat(id)).mtime!.getTime() };
  },
  async delete(id, _props, _state) {
    await Deno.remove(id);
  },
  async modifyPlan(_id, planType, nextProps, currentProps, currentState) {
    // If you decide you don't want to make any changes to the plan, just return a 204 No Content response.
    if (planType !== "update") {
      return;
    }

    // The most common use case for this endpoint is to tell tf if the resource
    // requires replacement (ie: create then delete) instead of an inline update.
    return { requiresReplacement: currentProps?.path !== nextProps.path };

    // Other use cases include returning a set of modifiedProps.
    // For example to provide default values for any unset props.
    // eg: return { modifiedProps: { content: nextProps?.content ?? "Hello World" } }

    // Or returning diagnostics, eg: for Destroy plans. As well as any other validation.
    // see: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#resource-destroy-plan-diagnostics
    // eg: return {
    //   diagnostics: [{
    //     severity: "warning",
    //     summary: "Resource Destruction Considerations",
    //     detail: `Applying this resource destruction will only remove the resource
    //             from the Terraform state and will not call the deletion API due to API limitations.
    //             Manually use the web interface to fully destroy this resource.`
    //   }]
    // }
  },
});
