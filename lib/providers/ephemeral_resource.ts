import { JSONRPCMethodNotFoundError } from "@yieldray/json-rpc-ts";
import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";
import { type Diagnostics, isDiagnostics } from "./diagnostics.ts";

/**
 * Defines the methods that must be implemented by an ephemeral resource provider.
 * Ephemeral resources have a lifecycle that begins when opened, can be renewed periodically,
 * and must be explicitly closed. They can maintain private state between operations.
 *
 * @template TProps - The type of the properties/configuration for the ephemeral resource.
 * @template TResult - The type of the data returned when the resource is opened.
 * @template TPrivateData - The type of private data maintained between lifecycle operations (optional).
 */
export type EphemeralResourceProviderMethods<TProps, TResult, TPrivateData = never> = {
  /**
   * Opens a new ephemeral resource with the provided properties.
   *
   * @param props - The properties/configuration for the ephemeral resource.
   * @returns A promise that resolves to an object containing the result data, an optional
   *          renewal timestamp, and optional private data to maintain between operations.
   *          Place any sensitive values under a "sensitive" key within the result to have
   *          them automatically marked as sensitive in Terraform.
   */
  open(props: TProps): Promise<
    Diagnostics | {
      result: TResult;
      /** Unix timestamp (in seconds) indicating when the resource should be renewed. */
      renewAt?: number;
      /** Private data to maintain between lifecycle operations (open, renew, close). */
      privateData?: TPrivateData;
    }
  >;

  /**
   * Renews an existing ephemeral resource. This method is optional.
   *
   * @param privateData - The private data from the previous open or renew operation.
   * @returns A promise that resolves to an object containing an optional renewal timestamp
   *          and updated private data.
   */
  renew?(privateData: TPrivateData): Promise<
    Diagnostics | {
      /** Unix timestamp (in seconds) indicating when the resource should be renewed again. */
      renewAt?: number;
      /** Updated private data to maintain for subsequent operations. */
      privateData?: TPrivateData;
    }
  >;

  /**
   * Closes an ephemeral resource and cleans up any associated resources. This method is optional.
   *
   * @param privateData - The private data from the previous open or renew operation.
   * @returns A promise that resolves when the resource is closed.
   */
  close?(privateData: TPrivateData): Promise<Diagnostics | void>;
};

/**
 * Base class for implementing Terraform ephemeral resource providers with JSON-RPC communication.
 * Ephemeral resources represent short-lived resources with a managed lifecycle (open, renew, close).
 *
 * @template TProps - The type of the properties/configuration for the ephemeral resource.
 * @template TResult - The type of the data returned when the resource is opened.
 * @template TPrivateData - The type of private data maintained between lifecycle operations.
 */
export class EphemeralResourceProvider<TProps, TResult, TPrivateData = never> extends BaseJsonRpcProvider {
  /**
   * Creates a new EphemeralResourceProvider instance.
   * @param providerMethods - The implementation of the ephemeral resource provider methods.
   */
  constructor(providerMethods: EphemeralResourceProviderMethods<TProps, TResult, TPrivateData>) {
    super(() => ({
      async open(params: { props: Record<string, unknown> }) {
        const result = await providerMethods.open(params.props as TProps);

        if (isDiagnostics(result)) return result;

        // deno-lint-ignore no-explicit-any
        const sensitiveResult = (result.result as any)?.sensitive;

        // deno-lint-ignore no-explicit-any
        const resultData = result.result as any;
        if (resultData && typeof resultData === "object" && "sensitive" in resultData) {
          delete resultData["sensitive"];
        }

        return { ...result, result: resultData, sensitiveResult };
      },
      async renew(params: { privateData: TPrivateData }) {
        if (!providerMethods.renew) throw new JSONRPCMethodNotFoundError();
        return await providerMethods.renew(params.privateData);
      },
      async close(params: { privateData: TPrivateData }) {
        if (!providerMethods.close) throw new JSONRPCMethodNotFoundError();
        const result = await providerMethods.close(params.privateData);
        if (isDiagnostics(result)) return result;
      },
    }));
  }
}

/**
 * Ephemeral resource provider with built-in Zod schema validation for properties, results, and private data.
 * Automatically validates all data flowing through the ephemeral resource lifecycle against the provided Zod schemas.
 *
 * @template TProps - A Zod schema type that defines the shape of the ephemeral resource properties.
 * @template TResult - A Zod schema type that defines the shape of the ephemeral resource result.
 * @template TPrivateData - A Zod schema type that defines the shape of the private data maintained between operations.
 */
export class ZodEphemeralResourceProvider<
  TProps extends z.ZodType,
  TResult extends z.ZodType,
  TPrivateData extends z.ZodType = never,
> extends EphemeralResourceProvider<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>> {
  /**
   * Creates a new ZodEphemeralResourceProvider instance with schema validation.
   *
   * @param propsSchema - The Zod schema used to validate ephemeral resource properties.
   * @param resultSchema - The Zod schema used to validate ephemeral resource results.
   * @param privateDataSchema - The Zod schema used to validate private data between operations.
   * @param providerMethods - The implementation of the ephemeral resource provider methods.
   */
  constructor(
    propsSchema: TProps,
    resultSchema: TResult,
    privateDataSchema: TPrivateData,
    providerMethods: EphemeralResourceProviderMethods<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>>,
  ) {
    const validatedMethods: EphemeralResourceProviderMethods<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>> =
      {
        async open(props) {
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
          const result = await providerMethods.open(propsParsed.data);

          // Catch any diagnostics and return them early
          if (isDiagnostics(result)) return result;

          // Validate the results
          const resultParsed = resultSchema.safeParse(result.result);
          const privateDataParsed = result.privateData ? privateDataSchema.safeParse(result.privateData) : null;
          if (!resultParsed.success || privateDataParsed?.success === false) {
            return {
              diagnostics: [
                ...(!resultParsed.success
                  ? resultParsed.error.issues.map((i) => ({
                    severity: "error",
                    summary: "Zod Validation Issue",
                    detail: i.message,
                    propPath: i.path.length > 0 ? ["result", ...i.path.map((_) => String(_))] : undefined,
                  }))
                  : []),
                ...(privateDataParsed?.success === false
                  ? privateDataParsed.error.issues.map((i) => ({
                    severity: "error",
                    summary: "Zod Validation Issue",
                    detail: i.message,
                  }))
                  : []),
              ],
            } as Diagnostics;
          }

          return {
            ...result,
            result: resultParsed.data,
            ...(privateDataParsed ? { privateData: privateDataParsed.data } : {}),
          };
        },
      };

    if (providerMethods.renew) {
      validatedMethods["renew"] = async (privateData) => {
        // Validate privateData
        const privateDataParsed = privateData ? privateDataSchema.safeParse(privateData) : null;
        if (privateDataParsed?.success === false) {
          return {
            diagnostics: privateDataParsed.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
            })),
          };
        }

        // Call the method with validated privateData
        const result = await providerMethods.renew!(
          // deno-lint-ignore no-explicit-any
          privateDataParsed ? privateDataParsed?.data : undefined as any,
        );

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Validate updated private data
        const updatedPrivateData = result.privateData ? privateDataSchema.safeParse(result.privateData) : null;
        if (updatedPrivateData?.success === false) {
          return {
            diagnostics: updatedPrivateData.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
            })),
          };
        }

        return {
          ...result,
          ...(updatedPrivateData ? { privateData: updatedPrivateData.data } : {}),
        };
      };
    }

    if (providerMethods.close) {
      validatedMethods["close"] = async (privateData) => {
        // Validate privateData
        const privateDataParsed = privateData ? privateDataSchema.safeParse(privateData) : null;
        if (privateDataParsed?.success === false) {
          return {
            diagnostics: privateDataParsed.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
            })),
          };
        }

        // Call the method with validated privateData
        const result = await providerMethods.close!(
          // deno-lint-ignore no-explicit-any
          privateDataParsed ? privateDataParsed?.data : undefined as any,
        );

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;
      };
    }

    super(validatedMethods);
  }
}
