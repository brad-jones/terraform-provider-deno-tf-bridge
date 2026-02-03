import { JSONRPCMethodNotFoundError } from "@yieldray/json-rpc-ts";
import type { z } from "@zod/zod";
import { BaseJsonRpcProvider } from "./base.ts";

export type EphemeralResourceProviderMethods<TProps, TResult, TPrivateData = never> = {
  open(props: TProps): Promise<{
    result: TResult;
    renewAt?: number;
    privateData?: TPrivateData;
  }>;
  renew?(privateData: TPrivateData): Promise<{
    renewAt?: number;
    privateData?: TPrivateData;
  }>;
  close?(privateData: TPrivateData): Promise<void>;
};

export class EphemeralResourceProvider<TProps, TResult, TPrivateData = never> extends BaseJsonRpcProvider {
  constructor(providerMethods: EphemeralResourceProviderMethods<TProps, TResult, TPrivateData>) {
    super(() => ({
      async open(params: { props: Record<string, unknown> }) {
        return await providerMethods.open(params.props as TProps);
      },
      async renew(params: { privateData: TPrivateData }) {
        if (!providerMethods.renew) {
          throw new JSONRPCMethodNotFoundError();
        }
        return await providerMethods.renew(params.privateData);
      },
      async close(params: { privateData: TPrivateData }) {
        if (!providerMethods.close) {
          throw new JSONRPCMethodNotFoundError();
        }
        return await providerMethods.close(params.privateData);
      },
    }));
  }
}

export class ZodEphemeralResourceProvider<
  TProps extends z.ZodType,
  TResult extends z.ZodType,
  TPrivateData extends z.ZodType = never,
> extends EphemeralResourceProvider<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>> {
  constructor(
    propsSchema: TProps,
    resultSchema: TResult,
    privateDataSchema: TPrivateData,
    providerMethods: EphemeralResourceProviderMethods<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>>,
  ) {
    const validatedMethods: EphemeralResourceProviderMethods<z.infer<TProps>, z.infer<TResult>, z.infer<TPrivateData>> =
      {
        async open(props) {
          const result = await providerMethods.open(propsSchema.parse(props));
          return {
            ...result,
            result: resultSchema.parse(result.result),
            privateData: privateDataSchema.parse(result.privateData),
          };
        },
      };

    if (providerMethods.renew) {
      validatedMethods["renew"] = async (privateData) => {
        const result = await providerMethods.renew!(privateDataSchema.parse(privateData));
        return {
          ...result,
          privateData: privateDataSchema.parse(result.privateData),
        };
      };
    }

    if (providerMethods.close) {
      validatedMethods["close"] = async (privateData) => {
        await providerMethods.close!(privateDataSchema.parse(privateData));
      };
    }

    super(validatedMethods);
  }
}
