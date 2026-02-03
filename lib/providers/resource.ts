import { JSONRPCMethodNotFoundError } from "@yieldray/json-rpc-ts";
import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

export type ResourceProviderMethods<TProps, TState, TID = string> = {
  create(props: TProps): Promise<{ id: TID; state: TState }>;
  read(id: TID, props: TProps): Promise<{ props: TProps; state: TState } | { exists: false }>;
  update(
    id: TID,
    nextProps: TProps,
    currentProps: TProps,
    currentState: TState,
  ): Promise<TState>;
  delete(id: TID, props: TProps, state: TState): Promise<void>;
  modifyPlan?(
    id: TID | null,
    planType: "create" | "update" | "delete",
    nextProps: TProps,
    currentProps: TProps | null,
    currentState: TState | null,
  ): Promise<
    | {
      modifiedProps?: TProps;
      diagnostics?: {
        severity: "error" | "warning";
        summary: string;
        detail: string;
        propName?: string;
      }[];
    }
    | { requiresReplacement: boolean }
    | undefined
  >;
};

export class ResourceProvider<TProps, TState, TID = string> extends BaseJsonRpcProvider {
  constructor(providerMethods: ResourceProviderMethods<TProps, TState, TID>) {
    super(() => ({
      async create(params: { props: Record<string, unknown> }) {
        return await providerMethods.create(params.props as TProps);
      },
      async read(params: { id: TID; props: Record<string, unknown> }) {
        return await providerMethods.read(params.id, params.props as TProps);
      },
      async update(
        params: {
          id: TID;
          nextProps: Record<string, unknown>;
          currentProps: Record<string, unknown>;
          currentState: Record<string, unknown>;
        },
      ) {
        const state = await providerMethods.update(
          params.id,
          params.nextProps as TProps,
          params.currentProps as TProps,
          params.currentState as TState,
        );
        return { state };
      },
      async delete(params: { id: TID; props: Record<string, unknown>; state: Record<string, unknown> }) {
        await providerMethods.delete(params.id, params.props as TProps, params.state as TState);
        return { done: true };
      },
      async modifyPlan(
        params: {
          id?: TID;
          planType: "create" | "update" | "delete";
          nextProps: Record<string, unknown>;
          currentProps?: Record<string, unknown>;
          currentState?: Record<string, unknown>;
        },
      ) {
        if (!providerMethods.modifyPlan) {
          throw new JSONRPCMethodNotFoundError();
        }

        const result = await providerMethods.modifyPlan(
          params?.id ?? null,
          params.planType,
          params.nextProps as TProps,
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

export class ZodResourceProvider<TProps extends z.ZodType, TState extends z.ZodType, TID = string>
  extends ResourceProvider<z.infer<TProps>, z.infer<TState>, TID> {
  constructor(
    propsSchema: TProps,
    stateSchema: TState,
    providerMethods: ResourceProviderMethods<z.infer<TProps>, z.infer<TState>, TID>,
  ) {
    const validatedMethods: ResourceProviderMethods<z.infer<TProps>, z.infer<TState>, TID> = {
      async create(props) {
        const { id, state } = await providerMethods.create(propsSchema.parse(props));
        return { id, state: stateSchema.parse(state) };
      },
      async read(id, props) {
        const result = await providerMethods.read(id, propsSchema.parse(props));
        if ("exists" in result) return result;
        return {
          props: propsSchema.parse(result.props),
          state: stateSchema.parse(result.state),
        };
      },
      async update(id, nextProps, currentProps, currentState) {
        return stateSchema.parse(
          await providerMethods.update(
            id,
            propsSchema.parse(nextProps),
            propsSchema.parse(currentProps),
            stateSchema.parse(currentState),
          ),
        );
      },
      async delete(id, props, state) {
        await providerMethods.delete(id, propsSchema.parse(props), stateSchema.parse(state));
      },
    };
    if (providerMethods.modifyPlan) {
      validatedMethods["modifyPlan"] = async (id, planType, nextProps, currentProps, currentState) => {
        const result = await providerMethods.modifyPlan!(
          id,
          planType,
          propsSchema.parse(nextProps),
          propsSchema.parse(currentProps),
          stateSchema.parse(currentState),
        );
        if (!result) return undefined;
        if ("requiresReplacement" in result) return result;
        return { ...result, modifiedProps: propsSchema.parse(result.modifiedProps) };
      };
    }
    super(validatedMethods);
  }
}
