# Error Handling with Diagnostics

This guide explains how error and warning handling works in terraform-provider-denobridge using the structured diagnostics system. It covers how to return errors from your Deno provider implementations, how they're processed by Terraform, and best practices for providing helpful feedback to users.

## Overview

The terraform-provider-denobridge uses a structured diagnostics system that allows provider implementations to return detailed error and warning messages to users. This system is significantly more user-friendly than relying on generic JSON-RPC errors or requiring users to set `TF_LOG=debug` to understand what went wrong.

### Key Features

- **Structured Error Messages**: Return errors and warnings as structured objects with severity, summary, and detail fields
- **Property Path Attribution**: Associate diagnostics with specific configuration properties for precise error reporting
- **Multiple Diagnostics**: Return multiple diagnostics in a single response (useful for validation)
- **Severity Levels**: Distinguish between errors (blocking) and warnings (informational)
- **Automatic Validation**: When using Zod schemas, validation errors are automatically converted to diagnostics

## Diagnostic Structure

Each diagnostic is a structured object with the following fields:

```typescript
interface Diagnostic {
  /** Severity level: "error" or "warning" */
  severity: "error" | "warning";

  /** Short, user-friendly summary of the issue */
  summary: string;

  /** Detailed explanation with context and suggestions */
  detail: string;

  /** Optional path to the specific property causing the issue */
  propPath?: string[];
}
```

### Field Descriptions

#### severity

The severity level determines how Terraform handles the diagnostic:

- **`"error"`**: Blocks the operation from completing. Terraform will fail the plan/apply and display the error to the user.
- **`"warning"`**: Informational only. The operation continues, but the warning is displayed to the user.

#### summary

A concise, user-facing description of the issue. This should be:

- Short (ideally one sentence)
- Clear and descriptive
- Written in sentence case
- End without a period

**Examples:**

- `"Invalid email format"`
- `"Property is deprecated"`
- `"Connection timeout"`

#### detail

A more detailed explanation that provides:

- Additional context about the issue
- Why the error occurred
- Suggestions for how to fix it
- Any relevant constraints or requirements

**Examples:**

- `"The email address 'user@' is not a valid format. Email addresses must include both a local part and a domain (e.g., user@example.com)."`
- `"The property 'old_field' is deprecated and will be removed in version 2.0. Please use 'new_field' instead."`
- `"Failed to connect to the API endpoint after 30 seconds. Check that the endpoint is accessible and your network allows outbound connections."`

#### propPath

An optional array of strings that specifies the path to the property that caused the issue. This allows Terraform to highlight the exact location in the configuration file where the problem exists.

The path segments follow this convention:

- **Root segment**: Usually `"props"`, `"nextProps"`, or the name of the top-level property
- **Object keys**: String names of nested properties
- **Array indices**: Numeric strings representing list positions

**Examples:**

```typescript
// Simple property: props.email
propPath: ["props", "email"];

// Nested property: props.config.timeout
propPath: ["props", "config", "timeout"];

// Array element: props.servers[0]
propPath: ["props", "servers", "0"];

// Deeply nested: props.servers[0].endpoints[1].url
propPath: ["props", "servers", "0", "endpoints", "1", "url"];
```

## Returning Diagnostics from Methods

All provider methods can optionally return diagnostics instead of their normal result. When diagnostics are returned, the method should return an object with a `diagnostics` array property.

### Resource Provider Examples

#### Returning Diagnostics from create()

```typescript
import { ResourceProvider } from "@brad-jones/terraform-provider-denobridge";

const provider = new ResourceProvider({
  async create(props) {
    // Validation example
    if (!props.name || props.name.trim() === "") {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Name is required",
          detail: "The 'name' property must be a non-empty string.",
          propPath: ["props", "name"],
        }],
      };
    }

    // Business logic validation
    if (props.replicas < 1 || props.replicas > 100) {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Invalid replica count",
          detail: "The number of replicas must be between 1 and 100.",
          propPath: ["props", "replicas"],
        }],
      };
    }

    // Normal successful creation
    return {
      id: "resource-id",
      state: { status: "active" },
    };
  },
  // Other methods...
});
```

#### Returning Multiple Diagnostics

You can return multiple diagnostics in a single response, which is particularly useful for validation scenarios:

```typescript
async create(props) {
  const diagnostics = [];

  // Validate multiple fields
  if (!props.email || !props.email.includes("@")) {
    diagnostics.push({
      severity: "error",
      summary: "Invalid email address",
      detail: "Email must be a valid address containing an @ symbol.",
      propPath: ["props", "email"]
    });
  }

  if (!props.password || props.password.length < 8) {
    diagnostics.push({
      severity: "error",
      summary: "Weak password",
      detail: "Password must be at least 8 characters long.",
      propPath: ["props", "password"]
    });
  }

  if (props.age < 18) {
    diagnostics.push({
      severity: "warning",
      summary: "Age verification recommended",
      detail: "Users under 18 may have limited access to certain features.",
      propPath: ["props", "age"]
    });
  }

  // Return all diagnostics if any were collected
  if (diagnostics.length > 0) {
    return { diagnostics };
  }

  // Otherwise proceed with creation
  return {
    id: generateId(),
    state: { created: Date.now() }
  };
}
```

#### Returning Warnings Alongside Success

Methods can return both their normal result AND diagnostics. This is useful for warnings that shouldn't block the operation:

```typescript
async create(props) {
  const result = {
    id: await createResource(props),
    state: { status: "created" }
  };

  // Add a warning if using deprecated configuration
  if (props.legacyMode) {
    return {
      ...result,
      diagnostics: [{
        severity: "warning",
        summary: "Legacy mode is deprecated",
        detail: "The 'legacyMode' option is deprecated and will be removed in v2.0. Consider migrating to the new configuration format.",
        propPath: ["props", "legacyMode"]
      }]
    };
  }

  return result;
}
```

#### Returning Diagnostics from read()

The `read()` method can return diagnostics to indicate issues with refreshing state:

```typescript
async read(id, props) {
  try {
    const resource = await fetchResource(id);

    if (!resource) {
      // Indicate the resource no longer exists
      return { exists: false };
    }

    return {
      props: resource.config,
      state: resource.state
    };
  } catch (error) {
    if (error instanceof PermissionError) {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Permission denied",
          detail: `Unable to read resource '${id}': ${error.message}. Verify that your credentials have read permissions.`
        }]
      };
    }
    throw error;
  }
}
```

#### Returning Diagnostics from update()

```typescript
async update(id, nextProps, currentProps, currentState) {
  // Check for unsupported updates
  if (nextProps.region !== currentProps.region) {
    return {
      diagnostics: [{
        severity: "error",
        summary: "Region cannot be changed",
        detail: "The 'region' property cannot be modified after resource creation. To change regions, the resource must be destroyed and recreated.",
        propPath: ["nextProps", "region"]
      }]
    };
  }

  // Perform the update
  const newState = await updateResource(id, nextProps);
  return newState;
}
```

#### Returning Diagnostics from delete()

```typescript
async delete(id, props, state) {
  try {
    await deleteResource(id);
  } catch (error) {
    if (error instanceof ResourceInUseError) {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Resource is in use",
          detail: `Cannot delete resource '${id}' because it is currently in use by other resources. Remove all dependencies before deleting.`
        }]
      };
    }
    throw error;
  }
}
```

### Data Source Provider Examples

Data sources can also return diagnostics:

```typescript
import { DatasourceProvider } from "@brad-jones/terraform-provider-denobridge";

const provider = new DatasourceProvider({
  async read(props) {
    // Validate query parameters
    if (!props.query) {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Query is required",
          detail: "The 'query' parameter must be provided to search for data.",
          propPath: ["props", "query"],
        }],
      };
    }

    const results = await searchData(props.query);

    if (results.length === 0) {
      return {
        diagnostics: [{
          severity: "warning",
          summary: "No results found",
          detail:
            `The query '${props.query}' returned no results. Verify that the query is correct and that matching data exists.`,
        }],
      };
    }

    return {
      result: results[0],
    };
  },
});
```

### Ephemeral Resource Provider Examples

Ephemeral resources can return diagnostics from `open()`, `renew()`, and `close()` methods:

```typescript
import { EphemeralResourceProvider } from "@brad-jones/terraform-provider-denobridge";

const provider = new EphemeralResourceProvider({
  async open(props) {
    try {
      const token = await generateToken(props.scope);

      return {
        result: { token },
        renewAt: Date.now() + 3600,
        privateData: { tokenId: token.id },
        diagnostics: [{
          severity: "warning",
          summary: "Token expires in 1 hour",
          detail:
            "The ephemeral token will expire in 1 hour. It will be automatically renewed if the operation takes longer.",
        }],
      };
    } catch (error) {
      return {
        diagnostics: [{
          severity: "error",
          summary: "Failed to generate token",
          detail: `Token generation failed: ${error.message}. Check your authentication credentials.`,
        }],
      };
    }
  },

  async renew(privateData) {
    const remaining = getRateLimit();

    const newToken = await renewToken(privateData.tokenId);

    if (remaining < 5) {
      return {
        renewAt: Date.now() + 3600,
        privateData: { tokenId: newToken.id },
        diagnostics: [{
          severity: "warning",
          summary: "Rate limit approaching",
          detail:
            `Only ${remaining} renewal requests remaining in this period. Consider reducing the frequency of token usage.`,
        }],
      };
    }

    return {
      renewAt: Date.now() + 3600,
      privateData: { tokenId: newToken.id },
    };
  },

  async close(privateData) {
    try {
      await revokeToken(privateData.tokenId);
    } catch (error) {
      return {
        diagnostics: [{
          severity: "warning",
          summary: "Token revocation failed",
          detail: `Unable to revoke token: ${error.message}. The token will expire naturally.`,
        }],
      };
    }
  },
});
```

### Action Provider Examples

Actions can return diagnostics to indicate partial failures or warnings:

```typescript
import { ActionProvider } from "@brad-jones/terraform-provider-denobridge";

const provider = new ActionProvider({
  async invoke(props) {
    const results = [];

    for (const target of props.targets) {
      try {
        await executeAction(target);
        results.push({ target, success: true });
      } catch (error) {
        results.push({ target, success: false, error: error.message });
      }
    }

    const failures = results.filter((r) => !r.success);

    if (failures.length > 0) {
      return {
        diagnostics: [{
          severity: "warning",
          summary: `${failures.length} of ${results.length} actions failed`,
          detail: `Failed targets: ${failures.map((f) => f.target).join(", ")}. Errors: ${
            failures.map((f) => f.error).join("; ")
          }`,
        }],
      };
    }
  },
});
```

## Automatic Validation with Zod

When using `ZodResourceProvider`, `ZodDatasourceProvider`, `ZodEphemeralResourceProvider`, or `ZodActionProvider`, validation errors are automatically converted to diagnostics. This eliminates the need to manually validate input and construct diagnostic objects.

### Example with ZodResourceProvider

```typescript
import { z } from "@zod/zod";
import { ZodResourceProvider } from "@brad-jones/terraform-provider-denobridge";

// Define schemas
const propsSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  email: z.string().email("Invalid email format"),
  age: z.number().int().min(0).max(150),
  tags: z.array(z.string()).optional(),
  config: z.object({
    enabled: z.boolean(),
    timeout: z.number().positive("Timeout must be positive"),
  }),
});

const stateSchema = z.object({
  id: z.string(),
  status: z.enum(["active", "inactive"]),
  createdAt: z.number(),
});

// Create provider with automatic validation
const provider = new ZodResourceProvider(
  propsSchema,
  stateSchema,
  {
    async create(props) {
      // props is already validated here!
      // If validation failed, diagnostics were returned automatically

      return {
        id: generateId(),
        state: {
          id: generateId(),
          status: "active",
          createdAt: Date.now(),
        },
      };
    },

    async read(id, props) {
      return {
        props: await fetchProps(id),
        state: await fetchState(id),
      };
    },

    async update(id, nextProps, currentProps, currentState) {
      // nextProps is validated automatically
      return await updateResource(id, nextProps);
    },

    async delete(id, props, state) {
      await deleteResource(id);
    },
  },
);
```

### Example Zod Validation Error

If a user provides invalid props, they'll automatically see helpful diagnostics:

**Terraform Configuration:**

```hcl
resource "denobridge_resource" "example" {
  path = "./provider.ts"
  props = {
    name = ""
    email = "invalid-email"
    age = -5
    config = {
      enabled = true
      timeout = -10
    }
  }
}
```

**Resulting Diagnostics:**

```
Error: Zod Validation Issue

  on main.tf line 4, in resource "denobridge_resource" "example":
   4:     name = ""

Name must not be empty

───────────────────────────────────────────────────────────

Error: Zod Validation Issue

  on main.tf line 5, in resource "denobridge_resource" "example":
   5:     email = "invalid-email"

Invalid email format

───────────────────────────────────────────────────────────

Error: Zod Validation Issue

  on main.tf line 6, in resource "denobridge_resource" "example":
   6:     age = -5

Number must be greater than or equal to 0

───────────────────────────────────────────────────────────

Error: Zod Validation Issue

  on main.tf line 9, in resource "denobridge_resource" "example":
   9:       timeout = -10

Timeout must be positive
```

### Combining Automatic and Manual Validation

You can combine Zod's automatic validation with your own manual diagnostics:

```typescript
const provider = new ZodResourceProvider(
  propsSchema,
  stateSchema,
  {
    async create(props) {
      // Zod validation has already passed at this point

      // But you can still add business logic validation
      const exists = await checkIfNameExists(props.name);
      if (exists) {
        return {
          diagnostics: [{
            severity: "error",
            summary: "Name already in use",
            detail: `A resource with the name '${props.name}' already exists. Choose a different name.`,
            propPath: ["props", "name"],
          }],
        };
      }

      // Proceed with creation
      return {
        id: generateId(),
        state: {
          id: generateId(),
          status: "active",
          createdAt: Date.now(),
        },
      };
    },
  },
);
```

## Best Practices

### 1. Be Specific and Actionable

**Bad:**

```typescript
{
  severity: "error",
  summary: "Invalid input",
  detail: "Something is wrong with your configuration"
}
```

**Good:**

```typescript
{
  severity: "error",
  summary: "Port number out of range",
  detail: "The port number must be between 1 and 65535. You provided: 99999",
  propPath: ["props", "port"]
}
```

### 2. Always Include propPath When Possible

Terraform uses `propPath` to highlight the exact location of the problem in the configuration file. This dramatically improves the user experience.

```typescript
// Without propPath - user has to search for the problem
{
  severity: "error",
  summary: "Invalid URL",
  detail: "One of the URLs is invalid"
}

// With propPath - Terraform highlights the exact line
{
  severity: "error",
  summary: "Invalid URL",
  detail: "The URL must start with http:// or https://",
  propPath: ["props", "endpoints", "2", "url"]
}
```

### 3. Use Warnings for Non-Blocking Issues

Not everything should be an error. Use warnings for:

- Deprecated features
- Performance concerns
- Best practice violations
- Informational messages

```typescript
{
  severity: "warning",
  summary: "Large batch size",
  detail: "Processing 10,000 items in a single batch may impact performance. Consider reducing batch size to 1,000 or less.",
  propPath: ["props", "batchSize"]
}
```

### 4. Provide Context and Suggestions

Help users fix the problem by providing context and actionable suggestions:

```typescript
{
  severity: "error",
  summary: "Authentication failed",
  detail: "Failed to authenticate with the API using the provided credentials. Check that your API key is valid and has not expired. You can generate a new API key at https://example.com/settings/keys",
  propPath: ["props", "apiKey"]
}
```

### 5. Handle Multiple Errors Gracefully

When validating multiple properties, collect all errors and return them together instead of failing on the first error:

```typescript
async create(props) {
  const diagnostics = [];

  // Validate all properties
  if (!props.field1) {
    diagnostics.push({
      severity: "error",
      summary: "Field 1 is required",
      detail: "...",
      propPath: ["props", "field1"]
    });
  }

  if (!props.field2) {
    diagnostics.push({
      severity: "error",
      summary: "Field 2 is required",
      detail: "...",
      propPath: ["props", "field2"]
    });
  }

  // Return all diagnostics at once
  if (diagnostics.length > 0) {
    return { diagnostics };
  }

  // Proceed with creation
  return { id: "...", state: {} };
}
```

### 6. Use Appropriate Severity Levels

- **Error**: Blocks execution, use for validation failures, permission issues, or any problem that prevents the operation from completing successfully
- **Warning**: Informational only, use for deprecation notices, performance concerns, or suggestions that don't prevent the operation from succeeding

### 7. Keep Summaries Concise

Summaries should be short and scannable. Save the details for the `detail` field:

**Bad:**

```typescript
{
  summary: "The timeout value you provided is too large and may cause the system to hang while waiting for a response from the upstream service, which could lead to poor user experience";
}
```

**Good:**

```typescript
{
  summary: "Timeout value too large",
  detail: "The timeout value of 300 seconds may cause the system to hang while waiting for a response. Consider reducing it to 30 seconds or less for better user experience."
}
```

## Debugging Diagnostics

If you need to debug your diagnostic handling:

1. **Enable Debug Logging**: Set `TF_LOG=debug` to see all JSON-RPC communication
2. **Check STDERR**: Any output to STDERR in your Deno script will appear in the debug logs
3. **Validate JSON Structure**: Ensure your diagnostics object matches the expected schema
4. **Test propPath**: Verify that your `propPath` arrays correctly represent the property location

## Summary

The diagnostics system in terraform-provider-denobridge provides a powerful way to communicate errors and warnings to users. By following the patterns and best practices in this guide, you can create provider implementations that give users clear, actionable feedback when things go wrong.

Key takeaways:

- All methods can optionally return diagnostics instead of their normal result
- Use structured diagnostic objects with severity, summary, detail, and propPath
- Leverage Zod providers for automatic validation with diagnostics
- Always include `propPath` to help Terraform highlight the exact problem location
- Use errors for blocking issues and warnings for informational messages
- Be specific, actionable, and helpful in your diagnostic messages
