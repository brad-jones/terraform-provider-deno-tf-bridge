// deno-lint-ignore-file no-explicit-any

import { JSONRPCMethodNotFoundError } from "@yieldray/json-rpc-ts";
import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";
import { type Diagnostics, isDiagnostics } from "./diagnostics.ts";

/** The return type for the modifyPlan method. */
type ModifyPlanReturn<TProps> = Promise<
  | {
    /** Modified properties to use instead of the originally planned properties. */
    modifiedProps?: TProps;
  }
  | {
    /** Whether the resource must be replaced (destroyed and recreated) instead of updated. */
    requiresReplacement: boolean;
  }
  | Diagnostics
  | undefined
>;

/**
 * Defines the methods for a stateful resource provider.
 * Resources maintain both configuration properties and runtime state.
 *
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TState - The type of the runtime state maintained by the resource.
 * @template TID - The type of the resource identifier (defaults to string).
 */
type StatefulResourceProviderMethods<TProps, TState, TID = string> = {
  /**
   * Creates a new resource with the provided properties.
   *
   * @param props - The properties/configuration for the new resource.
   * @returns A promise that resolves to an object containing the resource ID and initial state.
   */
  create(props: TProps): Promise<Diagnostics | { id: TID; state: TState }>;

  /**
   * Reads an existing resource by its ID and validates it against the expected properties.
   *
   * @param id - The identifier of the resource to read.
   * @param props - The expected properties/configuration of the resource.
   *                Props may not always exist, for example when importing resource,
   *                they are given on a best effort basis.
   * @returns A promise that resolves to the current properties and state if the resource exists,
   *          or an object with exists: false if the resource no longer exists.
   */
  read(id: TID, props: TProps | null): Promise<Diagnostics | { props: TProps; state: TState } | { exists: false }>;

  /**
   * Updates an existing resource with new properties.
   *
   * @param id - The identifier of the resource to update.
   * @param nextProps - The new properties/configuration to apply.
   * @param currentProps - The current properties/configuration before the update.
   * @param currentState - The current state before the update.
   * @returns A promise that resolves to the updated state.
   */
  update(
    id: TID,
    nextProps: TProps,
    currentProps: TProps,
    currentState: TState,
  ): Promise<Diagnostics | TState>;

  /**
   * Deletes an existing resource.
   *
   * @param id - The identifier of the resource to delete.
   * @param props - The current properties/configuration of the resource.
   * @param state - The current state of the resource.
   * @returns A promise that resolves when the resource is deleted.
   */
  delete(id: TID, props: TProps, state: TState): Promise<Diagnostics | void>;

  /**
   * Modifies a Terraform plan before execution. This method is optional and allows customizing
   * the planned changes, adding diagnostics, or indicating that a resource replacement is required.
   *
   * @param id - The identifier of the resource (null for create operations).
   * @param planType - The type of operation being planned: "create", "update", or "delete".
   * @param nextProps - The new properties/configuration after the planned change (null for delete operations).
   * @param currentProps - The current properties/configuration (null for create operations).
   * @param currentState - The current state (null for create operations).
   * @returns A promise that resolves to an object with modified properties and/or diagnostics,
   *          a replacement indicator, or undefined to accept the plan as-is.
   */
  modifyPlan?(
    id: TID | null,
    planType: "create" | "update" | "delete",
    nextProps: TProps | null,
    currentProps: TProps | null,
    currentState: TState | null,
  ): ModifyPlanReturn<TProps>;
};

/**
 * Defines the methods for a stateless resource provider.
 * Resources that only need an ID and props, without additional computed state.
 *
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TID - The type of the resource identifier (defaults to string).
 */
type StatelessResourceProviderMethods<TProps, TID = string> = {
  /**
   * Creates a new resource with the provided properties.
   *
   * @param props - The properties/configuration for the new resource.
   * @returns A promise that resolves to an object containing the resource ID.
   */
  create(props: TProps): Promise<Diagnostics | { id: TID }>;

  /**
   * Reads an existing resource by its ID and validates it against the expected properties.
   *
   * @param id - The identifier of the resource to read.
   * @param props - The expected properties/configuration of the resource.
   *                Props may not always exist, for example when importing resource,
   *                they are given on a best effort basis.
   * @returns A promise that resolves to the current properties if the resource exists,
   *          or an object with exists: false if the resource no longer exists.
   */
  read(id: TID, props: TProps | null): Promise<Diagnostics | { props: TProps } | { exists: false }>;

  /**
   * Updates an existing resource with new properties.
   *
   * @param id - The identifier of the resource to update.
   * @param nextProps - The new properties/configuration to apply.
   * @param currentProps - The current properties/configuration before the update.
   * @returns A promise that resolves when the update is complete.
   */
  update(
    id: TID,
    nextProps: TProps,
    currentProps: TProps,
  ): Promise<Diagnostics | void>;

  /**
   * Deletes an existing resource.
   *
   * @param id - The identifier of the resource to delete.
   * @param props - The current properties/configuration of the resource.
   * @returns A promise that resolves when the resource is deleted.
   */
  delete(id: TID, props: TProps): Promise<Diagnostics | void>;

  /**
   * Modifies a Terraform plan before execution. This method is optional and allows customizing
   * the planned changes, adding diagnostics, or indicating that a resource replacement is required.
   *
   * @param id - The identifier of the resource (null for create operations).
   * @param planType - The type of operation being planned: "create", "update", or "delete".
   * @param nextProps - The new properties/configuration after the planned change (null for delete operations).
   * @param currentProps - The current properties/configuration (null for create operations).
   * @returns A promise that resolves to an object with modified properties and/or diagnostics,
   *          a replacement indicator, or undefined to accept the plan as-is.
   */
  modifyPlan?(
    id: TID | null,
    planType: "create" | "update" | "delete",
    nextProps: TProps | null,
    currentProps: TProps | null,
  ): ModifyPlanReturn<TProps>;
};

/**
 * Defines the methods that must be implemented by a resource provider.
 * Resources can be created, read, updated, and deleted (CRUD operations).
 * They can optionally maintain runtime state in addition to their configuration properties.
 *
 * When `TState` is `void` (the default), the resource is stateless — it only needs an ID and props.
 * When `TState` is provided, the resource maintains additional computed state throughout its lifecycle.
 *
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TState - The type of the runtime state maintained by the resource (defaults to void for stateless resources).
 * @template TID - The type of the resource identifier (defaults to string).
 */
export type ResourceProviderMethods<TProps, TState = void, TID = string> = [TState] extends [void]
  ? StatelessResourceProviderMethods<TProps, TID>
  : StatefulResourceProviderMethods<TProps, TState, TID>;

/**
 * Base class for implementing Terraform resource providers with JSON-RPC communication.
 * Resources support full CRUD operations (create, read, update, delete) and can optionally
 * maintain additional computed state throughout their lifecycle.
 *
 * When `TState` is `void` (the default), the resource is stateless — only an ID and props are managed.
 * When `TState` is provided, the resource also maintains additional runtime state.
 *
 * @template TProps - The type of the properties/configuration for the resource.
 * @template TState - The type of the runtime state maintained by the resource (defaults to void for stateless resources).
 * @template TID - The type of the resource identifier (defaults to string).
 */
export class ResourceProvider<TProps, TState = void, TID = string> extends BaseJsonRpcProvider {
  /**
   * Creates a new ResourceProvider instance.
   * @param providerMethods - The implementation of the resource provider methods.
   */
  constructor(providerMethods: ResourceProviderMethods<TProps, TState, TID>) {
    super(() => ({
      async create(params: { props: Record<string, unknown> }) {
        return await providerMethods.create(params.props as TProps);
      },
      async read(params: { id: TID; props: Record<string, unknown> | null }) {
        return await providerMethods.read(params.id, params.props as TProps | null);
      },
      async update(
        params: {
          id: TID;
          nextProps: Record<string, unknown>;
          currentProps: Record<string, unknown>;
          currentState: Record<string, unknown>;
        },
      ) {
        const result = await providerMethods.update(
          params.id,
          params.nextProps as TProps,
          params.currentProps as TProps,
          params.currentState as TState,
        );
        if (isDiagnostics(result)) return result;
        return { state: result };
      },
      async delete(params: { id: TID; props: Record<string, unknown>; state: Record<string, unknown> }) {
        const result = await providerMethods.delete(params.id, params.props as TProps, params.state as TState);
        if (isDiagnostics(result)) return result;
        return { done: true };
      },
      async modifyPlan(
        params: {
          id?: TID;
          planType: "create" | "update" | "delete";
          nextProps?: Record<string, unknown>;
          currentProps?: Record<string, unknown>;
          currentState?: Record<string, unknown>;
        },
      ) {
        if (!providerMethods.modifyPlan) throw new JSONRPCMethodNotFoundError();

        const result = await providerMethods.modifyPlan(
          params?.id ?? null,
          params.planType,
          params.nextProps as TProps ?? null,
          params.currentProps as TProps ?? null,
          params.currentState as TState ?? null,
        );

        if (result) {
          return result;
        }

        return { noChanges: true };
      },
    }));
  }
}

/**
 * Resource provider with built-in Zod schema validation for properties and optionally state.
 * Automatically validates all data flowing through the resource lifecycle (create, read, update, delete)
 * against the provided Zod schemas.
 *
 * When only a props schema is provided (2-arg constructor), the resource is stateless.
 * When both props and state schemas are provided (3-arg constructor), the resource is stateful.
 *
 * @template TProps - A Zod schema type that defines the shape of the resource properties.
 * @template TState - A Zod schema type that defines the shape of the resource state (defaults to void for stateless resources).
 * @template TID - The type of the resource identifier (defaults to string).
 */
export class ZodResourceProvider<TProps extends z.ZodType, TState extends z.ZodType | void = void, TID = string>
  extends ResourceProvider<z.infer<TProps>, TState extends z.ZodType ? z.infer<TState> : void, TID> {
  /**
   * Creates a new stateless ZodResourceProvider instance with schema validation for props only.
   * @param propsSchema - The Zod schema used to validate resource properties.
   * @param providerMethods - The implementation of the resource provider methods (stateless).
   */
  constructor(
    propsSchema: TProps,
    providerMethods: ResourceProviderMethods<z.infer<TProps>, void, TID>,
  );

  /**
   * Creates a new stateful ZodResourceProvider instance with schema validation for props and state.
   * @param propsSchema - The Zod schema used to validate resource properties.
   * @param stateSchema - The Zod schema used to validate resource state.
   * @param providerMethods - The implementation of the resource provider methods (stateful).
   */
  constructor(
    propsSchema: TProps,
    stateSchema: TState & z.ZodType,
    providerMethods: ResourceProviderMethods<z.infer<TProps>, TState extends z.ZodType ? z.infer<TState> : never, TID>,
  );

  constructor(propsSchema: TProps, ...args: any[]) {
    const stateSchema: TState | undefined = args.length === 2 ? args[0] : undefined;
    const providerMethods: ResourceProviderMethods<z.infer<TProps>, z.infer<TState>, TID> = args.length === 2
      ? args[1]
      : args[0];

    const validatedMethods = {
      async create(props: any) {
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
        const result = await providerMethods.create(propsParsed.data);

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Validate the state
        if (stateSchema) {
          const stateParsed = stateSchema.safeParse((result as any).state);
          if (!stateParsed.success) {
            return {
              diagnostics: stateParsed.error.issues.map((i) => ({
                severity: "error",
                summary: "Zod Validation Issue",
                detail: i.message,
                propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
              })),
            };
          }

          return { id: result.id, state: stateParsed.data };
        }

        return { id: result.id };
      },
      async read(id: TID, props: any) {
        // Validate props
        const propsParsed = props ? propsSchema.safeParse(props) : undefined;
        if (propsParsed?.success === false) {
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
        const result = await providerMethods.read(id, propsParsed?.data ?? null);

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Catch the exists case and return it early
        if ("exists" in result) return result;

        // Validate the results
        if (stateSchema) {
          const resultPropsParsed = propsSchema.safeParse(result.props);
          const resultStateParsed = stateSchema.safeParse((result as any).state);
          if (!resultPropsParsed.success || !resultStateParsed.success) {
            return {
              diagnostics: [
                ...(!resultPropsParsed.success
                  ? resultPropsParsed.error.issues.map((i) => ({
                    severity: "error",
                    summary: "Zod Validation Issue",
                    detail: i.message,
                    propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                  }))
                  : []),
                ...(!resultStateParsed.success
                  ? resultStateParsed.error.issues.map((i) => ({
                    severity: "error",
                    summary: "Zod Validation Issue",
                    detail: i.message,
                    propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
                  }))
                  : []),
              ],
            } as Diagnostics;
          }

          return {
            props: resultPropsParsed.data,
            state: resultStateParsed.data,
          };
        }

        const resultPropsParsed = propsSchema.safeParse(result.props);
        if (!resultPropsParsed.success) {
          return {
            diagnostics: [
              ...(!resultPropsParsed.success
                ? resultPropsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
            ],
          } as Diagnostics;
        }

        return {
          props: resultPropsParsed.data,
        };
      },
      async update(id: TID, nextProps: any, currentProps: any, currentState: any) {
        // Validate props
        const propsParsed = propsSchema.safeParse(nextProps);
        const currentPropsParsed = propsSchema.safeParse(currentProps);
        const currentStateParsed = stateSchema ? stateSchema.safeParse(currentState) : undefined;
        if (
          !propsParsed.success || !currentPropsParsed.success || (currentStateParsed && !currentStateParsed.success)
        ) {
          return {
            diagnostics: [
              ...(!propsParsed.success
                ? propsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
              ...(!currentPropsParsed.success
                ? currentPropsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
              ...((currentStateParsed && !currentStateParsed.success)
                ? currentStateParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
            ],
          } as Diagnostics;
        }

        // Call the method with validated props
        const result = await providerMethods.update(
          id,
          propsParsed.data,
          currentPropsParsed.data,
          currentStateParsed ? currentStateParsed.data as any : undefined,
        );

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Validate the state
        if (stateSchema) {
          const stateParsed = stateSchema.safeParse(result);
          if (!stateParsed.success) {
            return {
              diagnostics: stateParsed.error.issues.map((i) => ({
                severity: "error",
                summary: "Zod Validation Issue",
                detail: i.message,
                propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
              })),
            };
          }
          return stateParsed.data;
        }
      },
      async delete(id: TID, props: any, state: any) {
        // Validate props
        const propsParsed = propsSchema.safeParse(props);
        const stateParsed = stateSchema ? stateSchema.safeParse(state) : undefined;
        if (!propsParsed.success || (stateParsed && !stateParsed.success)) {
          return {
            diagnostics: [
              ...(!propsParsed.success
                ? propsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
              ...((stateParsed && !stateParsed.success)
                ? stateParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
            ],
          } as Diagnostics;
        }

        // Call the method with validated props
        const result = await providerMethods.delete(id, propsParsed.data, stateParsed?.data as any);

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;
      },
    };
    if (providerMethods.modifyPlan) {
      (validatedMethods as any)["modifyPlan"] = async (
        id: TID,
        planType: any,
        nextProps: any,
        currentProps: any,
        currentState: any,
      ) => {
        // Validate props
        const nextPropsParsed = nextProps ? propsSchema.safeParse(nextProps) : undefined;
        const currentPropsParsed = currentProps ? propsSchema.safeParse(currentProps) : undefined;
        const currentStateParsed = currentState && stateSchema ? stateSchema.safeParse(currentState) : undefined;
        if (
          nextPropsParsed?.success === false || currentPropsParsed?.success === false ||
          currentStateParsed?.success === false
        ) {
          return {
            diagnostics: [
              ...(nextPropsParsed?.success === false
                ? nextPropsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
              ...(currentPropsParsed?.success === false
                ? currentPropsParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
              ...(currentStateParsed?.success === false
                ? currentStateParsed.error.issues.map((i) => ({
                  severity: "error",
                  summary: "Zod Validation Issue",
                  detail: i.message,
                  propPath: i.path.length > 0 ? ["state", ...i.path.map((_) => String(_))] : undefined,
                }))
                : []),
            ],
          } as Diagnostics;
        }

        // Call the method with validated props
        const result = await providerMethods.modifyPlan!(
          id,
          planType,
          nextPropsParsed ? nextPropsParsed.data : null,
          currentPropsParsed ? currentPropsParsed.data : null,
          currentStateParsed ? currentStateParsed.data as any : null,
        );

        // Bail out early if there are no modifications needed
        if (!result) return undefined;

        // Catch any diagnostics and return them early
        if (isDiagnostics(result)) return result;

        // Catch the requiresReplacement case
        if ("requiresReplacement" in result) return result;

        // Validate the modified props
        const modifiedPropsParsed = result.modifiedProps ? propsSchema.safeParse(result.modifiedProps) : undefined;
        if (modifiedPropsParsed?.success === false) {
          return {
            diagnostics: modifiedPropsParsed.error.issues.map((i) => ({
              severity: "error",
              summary: "Zod Validation Issue",
              detail: i.message,
              propPath: i.path.length > 0 ? ["props", ...i.path.map((_) => String(_))] : undefined,
            })),
          };
        }

        return { ...result, modifiedProps: modifiedPropsParsed?.data };
      };
    }
    super(validatedMethods as any);
  }
}
