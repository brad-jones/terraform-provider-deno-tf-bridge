import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

export interface DatasourceProviderMethods<TProps, TResult> {
  read(props: TProps): Promise<TResult>;
}

export class DatasourceProvider<TProps, TResult> extends BaseJsonRpcProvider {
  constructor(providerMethods: DatasourceProviderMethods<TProps, TResult>) {
    super(() => ({
      async read(params: { props: unknown }) {
        return { result: await providerMethods.read(params.props as TProps) };
      },
    }));
  }
}

export class ZodDatasourceProvider<TProps extends z.ZodType, TResult extends z.ZodType>
  extends DatasourceProvider<z.infer<TProps>, z.infer<TResult>> {
  constructor(
    propsSchema: TProps,
    resultSchema: TResult,
    providerMethods: DatasourceProviderMethods<z.infer<TProps>, z.infer<TResult>>,
  ) {
    super({
      async read(props) {
        return resultSchema.parse(
          await providerMethods.read(
            propsSchema.parse(props),
          ),
        );
      },
    });
  }
}
