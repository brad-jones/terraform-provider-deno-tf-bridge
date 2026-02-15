import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";
import { type Diagnostics, isDiagnostics } from "./diagnostics.ts";

/**
 * Defines the methods that must be implemented by a datasource provider.
 * Datasources are read-only resources that fetch data from external sources.
 *
 * @template TProps - The type of the properties/configuration for the datasource.
 * @template TResult - The type of the data returned by the datasource.
 */
export interface DatasourceProviderMethods<TProps, TResult> {
  /**
   * Reads data from the datasource based on the provided properties.
   *
   * @param props - The properties/configuration for the datasource read operation.
   * @returns A promise that resolves to the data fetched from the datasource.
   */
  read(props: TProps): Promise<Diagnostics | TResult>;
}

/**
 * Base class for implementing Terraform datasource providers with JSON-RPC communication.
 * Datasources are read-only and used to fetch data from external sources during Terraform operations.
 *
 * @template TProps - The type of the properties/configuration for the datasource.
 * @template TResult - The type of the data returned by the datasource.
 */
export class DatasourceProvider<TProps, TResult> extends BaseJsonRpcProvider {
  /**
   * Creates a new DatasourceProvider instance.
   * @param providerMethods - The implementation of the datasource provider methods.
   */
  constructor(providerMethods: DatasourceProviderMethods<TProps, TResult>) {
    super(() => ({
      async read(params: { props: unknown }) {
        const result = await providerMethods.read(params.props as TProps);
        if (isDiagnostics(result)) return result;

        // deno-lint-ignore no-explicit-any
        const sensitiveResult = (result as any)?.sensitive;

        // deno-lint-ignore no-explicit-any
        const resultData = result as any;
        if (resultData && typeof resultData === "object" && "sensitive" in resultData) {
          delete resultData["sensitive"];
        }

        return { result: resultData, sensitiveResult };
      },
    }));
  }
}

/**
 * Datasource provider with built-in Zod schema validation for both properties and results.
 * Automatically validates incoming properties and outgoing results against the provided Zod schemas.
 *
 * @template TProps - A Zod schema type that defines the shape of the datasource properties.
 * @template TResult - A Zod schema type that defines the shape of the datasource result.
 */
export class ZodDatasourceProvider<TProps extends z.ZodType, TResult extends z.ZodType>
  extends DatasourceProvider<z.infer<TProps>, z.infer<TResult>> {
  /**
   * Creates a new ZodDatasourceProvider instance with schema validation.
   *
   * @param propsSchema - The Zod schema used to validate datasource properties.
   * @param resultSchema - The Zod schema used to validate datasource results.
   * @param providerMethods - The implementation of the datasource provider methods.
   */
  constructor(
    propsSchema: TProps,
    resultSchema: TResult,
    providerMethods: DatasourceProviderMethods<z.infer<TProps>, z.infer<TResult>>,
  ) {
    super({
      async read(props) {
        // Validate props
        const propsParsed = propsSchema.safeParse(props);
        if (!propsParsed.success) {
          return {
            diagnostics: propsParsed.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
              propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
            })),
          };
        }

        // Call the method with validated props
        const result = await providerMethods.read(propsParsed.data);

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Validate the results
        const resultParsed = resultSchema.safeParse(result);
        if (!resultParsed.success) {
          return {
            diagnostics: resultParsed.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
              propPath: i.path.length > 0 ? ["result", ...i.path.map((_) => String(_))] : undefined,
            })),
          };
        }

        return resultParsed.data;
      },
    });
  }
}
