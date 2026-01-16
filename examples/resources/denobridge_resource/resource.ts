import { Hono } from "hono";

const app = new Hono();

app.get("/health", (c) => {
  // The provider will poll this endpoint until it gets a good response before
  // attempting to call any of the other lifecycle endpoints
  return c.body(null, 204);
});

app.post("/create", async (c) => {
  // Create is given a set of props that should contain all required
  // information to create a new instance of the resource.
  const body = await c.req.json();
  const { path, content } = body.props;

  // In this case we are just creating a file
  await Deno.writeTextFile(path, content);

  return c.json({
    // At minimum, the create endpoint must return an id that uniquely identifies this resource.
    id: path,

    // Optional additional state that is only known after the resource is created can be returned in the state object.
    state: {
      mtime: (await Deno.stat(path)).mtime!.getTime(),
    },
  });
});

app.post("/read", async (c) => {
  // Read is given the id of a previously created resource along with it's props.
  // In this case the id is the same as the file path so we don't really need the
  // props to update the current state.
  const body = await c.req.json();
  const { id } = body;
  // const { path } = body.props; id === path

  // The read methods job is to get the current state of an existing resource.
  // And return an updated set of props & optional additional state that may
  // have changed (by outside actors), since the resource was initially created
  // (or last updated).
  try {
    const content = await Deno.readTextFile(id);
    return c.json({
      props: { path: id, content },
      state: {
        mtime: (await Deno.stat(id)).mtime!.getTime(),
      },
    });
  } catch (e) {
    if (e instanceof Deno.errors.NotFound) {
      // In the event we can no longer locate the resource we should signal
      // to tf to remove the resource from state. This will cause tf to re-create
      // the resource on the next plan.
      return c.json({ exists: false });
    }
    throw e;
  }
});

app.post("/update", async (c) => {
  // Update is given the id of a previously created resource along with the
  // current props & state (this will either have come from the saved tf state
  // or more likely be the refreshed data returned from the /read endpoint),
  // as well as the next set of props to update the resource to.
  const body = await c.req.json();
  const { id } = body;
  const { path: currentPath } = body.currentProps;
  const { mtime: currentMTime } = body.currentState;
  const { path: nextPath, content: nextContent } = body.nextProps;

  // In this case an in place update is not supported for a file when the path
  // changes because that would mean the ID would change too & tf can't deal with that easily.
  // So we return an error here but if an update wouldn't result in a new ID, there is no need to return such an error.
  // Or implement the /modify-plan endpoint, which should have caught this & told tf to do a replacement instead (ie: create then delete).
  if (nextPath !== currentPath) {
    return c.json({ error: "Cannot change file path - requires resource replacement" }, 422);
  }

  // Perform the update
  await Deno.writeTextFile(currentPath, nextContent);

  // Return the updated state
  return c.json({ state: { mtime: (await Deno.stat(currentPath)).mtime!.getTime() } });

  // NB: If the update doesn't result in any state change (ie: there is simply no additional computed props to return)
  // Then you can return a 204 Not Content response to signal the update was successfully applied.
  // eg: return c.body(null, 204);
});

app.post("/delete", async (c) => {
  // Delete is given the id of a previously created resource along with it's props & any additional computed state.
  const body = await c.req.json();
  const { id } = body;
  const { path } = body.props;
  const { mtime } = body.state;

  // Perform the delete
  await Deno.remove(path);

  // And signal success a 204 Not Content response.
  return c.body(null, 204);
});

// This endpoint is optional, if your server returns a 404, the provider will just carry on without making any plan modifications.
app.post("/modify-plan", async (c) => {
  // The ModifyPlan method is given:
  //  - The id of the resource (which could be empty if the resource is yet to be created)
  //  - The type of plan that we are modifying (create, update or delete)
  //  - nextProps will be set for create & update
  //  - currentProps will be set for update & delete
  //  - currentState will be set for update & delete
  const body = await c.req.json();
  const { planType } = body;

  // If you decide you don't want to make any changes to the plan, just return a 204 No Content response.
  if (planType !== "update") {
    return c.body(null, 204);
  }

  // The most common use case for this endpoint is to tell tf if the resource
  // requires replacement (ie: create then delete) instead of an inline update.
  const { currentProps, nextProps } = body;
  return c.json({ requiresReplacement: currentProps.path !== nextProps.path });

  // Other use cases include returning a set of modifiedProps.
  // For example to provide default values for any unset props.
  // eg: return c.json({ modifiedProps: { content: nextProps?.content ?? "Hello World" } })

  // Or returning diagnostics, eg: for Destroy plans. As well as any other validation.
  // see: https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#resource-destroy-plan-diagnostics
  // eg: return c.json({
  //   diagnostics: [{
  //     severity: "warning",
  //     summary: "Resource Destruction Considerations",
  //     detail: `Applying this resource destruction will only remove the resource
  //             from the Terraform state and will not call the deletion API due to API limitations.
  //             Manually use the web interface to fully destroy this resource.`
  //   }]
  // })
});

export default app satisfies Deno.ServeDefaultExport;
