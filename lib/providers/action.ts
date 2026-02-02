import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

export type ActionProviderMethods<TProps> = {
  invoke(props: TProps, progressCallback: (message: string) => Promise<void>): Promise<void>;
};

type RemoteMethods = {
  invokeProgress(params: { message: string }): void;
};

export class ActionProvider<TProps> extends BaseJsonRpcProvider<RemoteMethods> {
  constructor(providerMethods: ActionProviderMethods<TProps>) {
    super((client) => ({
      async invoke(params: { props: Record<string, unknown> }) {
        await providerMethods.invoke(
          params.props as TProps,
          (message: string) => client.notify("invokeProgress", { message }),
        );
        return { done: true };
      },
    }));
  }
}

export class ZodActionProvider<
  TProps extends z.ZodType,
> extends ActionProvider<z.infer<TProps>> {
  constructor(
    propsSchema: TProps,
    providerMethods: ActionProviderMethods<z.infer<TProps>>,
  ) {
    super({
      async invoke(props, progressCallback) {
        await providerMethods.invoke(propsSchema.parse(props), progressCallback);
      },
    });
  }
}
