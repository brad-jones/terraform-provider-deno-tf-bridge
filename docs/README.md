# DenoBridge API Specifications

This directory contains OpenAPI specifications for the HTTP API contract
between the Terraform provider and Deno TypeScript implementations.

## Overview

The denobridge provider allows TypeScript developers to write Terraform resources,
data sources, ephemeral resources, and actions using Deno. The provider starts your Deno
script as an HTTP server and communicates with it via a well-defined HTTP API.

## Specifications

### [denobridge_resource.yaml](./denobridge_resource.yaml)

Full CRUD lifecycle management for Terraform managed resources.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /create` - Create a new resource instance
- `POST /read` - Read the current state of a resource
- `POST /update` - Update an existing resource
- `POST /delete` - Delete a resource

**Optional Endpoints:**

- `POST /modify-plan` - Modify Terraform plans (e.g., signal replacement needed)

**Use Cases:**

- Managing infrastructure resources
- Creating and managing files
- Provisioning external services

### [denobridge_datasource.yaml](./denobridge_datasource.yaml)

Read-only data fetching for Terraform data sources.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /read` - Fetch data based on input properties

**Use Cases:**

- Fetching external data
- Performing computations (hashing, encoding, etc.)
- Querying APIs or databases

### [denobridge_ephemeral_resource.yaml](./denobridge_ephemeral_resource.yaml)

Short-lived resources that exist only during Terraform operations.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /open` - Open/create the ephemeral resource

**Optional Endpoints:**

- `POST /renew` - Renew expiring resources
- `POST /close` - Clean up when no longer needed

**Use Cases:**

- Temporary credentials or tokens
- Time-limited access grants
- Ephemeral compute resources

### [denobridge_action.yaml](./denobridge_action.yaml)

Operations that don't manage resources but perform actions.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /invoke` - Execute the action with streaming progress

**Use Cases:**

- Sending notifications
- Triggering webhooks
- Running validation tasks
- Executing external workflows

## Common Patterns

### Health Check

All resource types must implement a `/health` endpoint that returns a 204 No Content response.
The provider polls this endpoint until it succeeds before making any other requests.

```typescript
app.get("/health", (c) => {
  return c.body(null, 204);
});
```

### Error Handling

Return appropriate HTTP status codes:

- `200` - Success with content
- `204` - Success without content
- `404` - Endpoint not implemented (for optional endpoints)
- `422` - Unprocessable entity (validation errors)
- `4XX` - Client errors
- `5XX` - Server errors

For error responses, return a JSON object with an `error` field:

```json
{
  "error": "Cannot change file path - requires resource replacement"
}
```

### Dynamic Properties

All `props` and `state` fields are dynamic objects that can contain any JSON-serializable data structure.
Design your API contract based on your resource's needs.

### Private Data (Ephemeral Resources)

Ephemeral resources can return private data from `/open` and `/renew` that will
be passed to subsequent `/renew` and `/close` calls. This is useful for storing
credentials or state needed for cleanup.

## Examples

See the test files in the repository for complete working examples:

- [resource_test.ts](../internal/provider/resource_test.ts) - File management resource
- [datasource_test.ts](../internal/provider/datasource_test.ts) - SHA-256 hashing
- [ephemeral_resource_test.ts](../internal/provider/ephemeral_resource_test.ts) - UUID generation
- [action_test.ts](../internal/provider/action_test.ts) - File writing with progress

## Development Workflow

1. Choose the appropriate resource type for your use case
2. Review the OpenAPI specification for required and optional endpoints
3. Implement the HTTP server using Hono or another Deno-compatible framework
4. Start with the `/health` endpoint to verify basic connectivity
5. Implement the core lifecycle endpoints
6. Test with Terraform configuration
7. Add optional endpoints as needed (e.g., `/modify-plan`, `/renew`, `/close`)

## Deno Permissions

All resource types support configuring Deno runtime permissions via the `permissions` block in Terraform configuration:

```hcl
resource "denobridge_resource" "example" {
  path = "./script.ts"

  permissions {
    allow = ["read", "write", "net"]
  }
}
```

## Tools

You can use these OpenAPI specifications with various tools:

- **Swagger UI** - Interactive API documentation
- **OpenAPI Generator** - Generate client/server code
- **Postman** - API testing and development
- **VS Code Extensions** - OpenAPI/Swagger viewers

## Contributing

When making changes to the API contract, please update the relevant OpenAPI specification and increment the version number.
