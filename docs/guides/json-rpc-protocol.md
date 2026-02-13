# Low-Level JSON-RPC Protocol

This guide documents the low-level JSON-RPC 2.0 protocol used for communication between the Go-based Terraform provider and Deno child processes. This documentation is intended for developers who want to implement the TypeScript/Deno side from scratch, without using the `@brad-jones/terraform-provider-denobridge` JSR package.

## Overview

The terraform-provider-denobridge uses a bidirectional JSON-RPC 2.0 protocol over standard input/output (stdin/stdout) streams to communicate between the Go Terraform provider and Deno child processes. Each provider type (Resource, Data Source, Ephemeral Resource, and Action) implements a specific set of JSON-RPC methods.

### Transport Layer

- **Protocol**: JSON-RPC 2.0 (as defined in [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification))
- **Transport**: Newline-delimited JSON over stdin/stdout
- **Direction**: Bidirectional (both parties can act as client and server)
- **Encoding**: UTF-8 text with each JSON-RPC message terminated by a newline (`\n`)

### Message Format

All messages follow the JSON-RPC 2.0 specification:

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "methodName",
  "params": { "key": "value" },
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": { "key": "value" },
  "id": 1
}
```

**Error Response:**

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found"
  },
  "id": 1
}
```

**Notification (no response expected):**

```json
{
  "jsonrpc": "2.0",
  "method": "methodName",
  "params": { "key": "value" }
}
```

## Common Methods

These methods are available for all provider types and are automatically provided by the base implementation:

### health

**Direction**: Go → Deno

A health check method used by the Go provider to verify that the Deno process is responsive.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "health",
  "params": {},
  "id": 1
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "ok": true
  },
  "id": 1
}
```

#### OpenRPC Schema

```json
{
  "name": "health",
  "description": "Health check to verify the Deno process is responsive",
  "params": [],
  "result": {
    "name": "healthResult",
    "schema": {
      "type": "object",
      "properties": {
        "ok": {
          "type": "boolean",
          "description": "Always true when responding"
        }
      },
      "required": ["ok"]
    }
  }
}
```

### shutdown

**Direction**: Go → Deno

Signals the Deno process to perform a graceful shutdown.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "shutdown",
  "params": {},
  "id": 2
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {},
  "id": 2
}
```

#### OpenRPC Schema

```json
{
  "name": "shutdown",
  "description": "Signals graceful shutdown of the Deno process",
  "params": [],
  "result": {
    "name": "shutdownResult",
    "schema": {
      "type": "object"
    }
  }
}
```

## Resource Provider

Resources represent managed infrastructure objects with a full lifecycle (create, read, update, delete).

### create

**Direction**: Go → Deno

Creates a new resource instance.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "create",
  "params": {
    "props": {
      "// User-defined configuration properties": "..."
    }
  },
  "id": 3
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "id": "resource-unique-identifier",
    "state": {
      "// Computed state values": "..."
    },
    "sensitiveState": {
      "// Sensitive computed state values": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Property normalized",
        "detail": "The value was normalized to a standard format",
        "propPath": ["props", "example_property"]
      }
    ]
  },
  "id": 3
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "create",
  "description": "Creates a new resource instance",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "props": {
            "type": "object",
            "description": "User-defined configuration properties for the resource"
          }
        },
        "required": ["props"]
      }
    }
  ],
  "result": {
    "name": "createResult",
    "schema": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "Unique identifier for the created resource"
        },
        "state": {
          "type": "object",
          "description": "Computed state values for the resource"
        },
        "sensitiveState": {
          "type": "object",
          "description": "Sensitive computed state values for the resource (marked as sensitive in Terraform)"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["id", "state"]
    }
  }
}
```

### read

**Direction**: Go → Deno

Reads the current state of a resource instance. Can indicate resource no longer exists.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "read",
  "params": {
    "id": "resource-unique-identifier",
    "props": {
      "// User-defined configuration properties": "..."
    }
  },
  "id": 4
}
```

#### Response (Resource Exists)

```json
{
  "jsonrpc": "2.0",
  "result": {
    "props": {
      "// Refreshed configuration properties": "..."
    },
    "state": {
      "// Refreshed computed state": "..."
    },
    "sensitiveState": {
      "// Refreshed sensitive computed state": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Deprecated property",
        "detail": "The property 'old_field' is deprecated and will be removed in a future version",
        "propPath": ["props", "old_field"]
      }
    ]
  },
  "id": 4
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### Response (Resource Doesn't Exist)

```json
{
  "jsonrpc": "2.0",
  "result": {
    "exists": false
  },
  "id": 4
}
```

#### OpenRPC Schema

```json
{
  "name": "read",
  "description": "Reads the current state of a resource instance",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string",
            "description": "Unique identifier of the resource to read"
          },
          "props": {
            "type": "object",
            "description": "Current configuration properties"
          }
        },
        "required": ["id", "props"]
      }
    }
  ],
  "result": {
    "name": "readResult",
    "schema": {
      "oneOf": [
        {
          "type": "object",
          "properties": {
            "props": {
              "type": "object",
              "description": "Refreshed configuration properties"
            },
            "state": {
              "type": "object",
              "description": "Refreshed computed state"
            },
            "sensitiveState": {
              "type": "object",
              "description": "Refreshed sensitive computed state"
            },
            "diagnostics": {
              "type": "array",
              "description": "Optional warnings or errors to display to the user",
              "items": {
                "type": "object",
                "properties": {
                  "severity": {
                    "type": "string",
                    "enum": ["error", "warning"],
                    "description": "Diagnostic severity level"
                  },
                  "summary": {
                    "type": "string",
                    "description": "Short description of the diagnostic"
                  },
                  "detail": {
                    "type": "string",
                    "description": "Additional context about the diagnostic"
                  },
                  "propPath": {
                    "type": "array",
                    "items": {
                      "type": "string"
                    },
                    "description": "Path to the property this diagnostic relates to"
                  }
                },
                "required": ["severity", "summary", "detail"]
              }
            }
          },
          "required": ["props", "state"]
        },
        {
          "type": "object",
          "properties": {
            "exists": {
              "type": "boolean",
              "const": false,
              "description": "Indicates the resource no longer exists"
            }
          },
          "required": ["exists"]
        }
      ]
    }
  }
}
```

### update

**Direction**: Go → Deno

Updates an existing resource instance with new configuration.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "update",
  "params": {
    "id": "resource-unique-identifier",
    "nextProps": {
      "// New desired configuration": "..."
    },
    "currentProps": {
      "// Current configuration": "..."
    },
    "currentState": {
      "// Current computed state": "..."
    },
    "currentSensitiveState": {
      "// Current sensitive computed state": "..."
    }
  },
  "id": 5
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "state": {
      "// Updated computed state": "..."
    },
    "sensitiveState": {
      "// Updated sensitive computed state": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Update may take time",
        "detail": "This update operation may take several minutes to complete"
      }
    ]
  },
  "id": 5
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "update",
  "description": "Updates an existing resource instance",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string",
            "description": "Unique identifier of the resource to update"
          },
          "nextProps": {
            "type": "object",
            "description": "New desired configuration properties"
          },
          "currentProps": {
            "type": "object",
            "description": "Current configuration properties before the update"
          },
          "currentState": {
            "type": "object",
            "description": "Current computed state before the update"
          },
          "currentSensitiveState": {
            "type": "object",
            "description": "Current sensitive computed state before the update"
          }
        },
        "required": ["id", "nextProps", "currentProps", "currentState"]
      }
    }
  ],
  "result": {
    "name": "updateResult",
    "schema": {
      "type": "object",
      "properties": {
        "state": {
          "type": "object",
          "description": "Updated computed state after the update"
        },
        "sensitiveState": {
          "type": "object",
          "description": "Updated sensitive computed state after the update"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["state"]
    }
  }
}
```

### delete

**Direction**: Go → Deno

Deletes a resource instance.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "delete",
  "params": {
    "id": "resource-unique-identifier",
    "props": {
      "// Configuration properties": "..."
    },
    "state": {
      "// Current computed state": "..."
    },
    "sensitiveState": {
      "// Current sensitive computed state": "..."
    }
  },
  "id": 6
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "done": true,
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Resource cleanup incomplete",
        "detail": "Some associated resources may need to be manually cleaned up"
      }
    ]
  },
  "id": 6
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "delete",
  "description": "Deletes a resource instance",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "id": {
            "type": "string",
            "description": "Unique identifier of the resource to delete"
          },
          "props": {
            "type": "object",
            "description": "Configuration properties"
          },
          "state": {
            "type": "object",
            "description": "Current computed state"
          },
          "sensitiveState": {
            "type": "object",
            "description": "Current sensitive computed state"
          }
        },
        "required": ["id", "props", "state"]
      }
    }
  ],
  "result": {
    "name": "deleteResult",
    "schema": {
      "type": "object",
      "properties": {
        "done": {
          "type": "boolean",
          "description": "Must be true to indicate successful deletion"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["done"]
    }
  }
}
```

### modifyPlan (Optional)

**Direction**: Go → Deno

Allows the resource to modify planned values before apply or indicate that a resource replacement is required. This method is optional and may return a "Method not found" error if not implemented.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "modifyPlan",
  "params": {
    "id": "resource-unique-identifier",
    "planType": "update",
    "nextProps": {
      "// Proposed new configuration": "..."
    },
    "currentProps": {
      "// Current configuration": "..."
    },
    "currentState": {
      "// Current computed state": "..."
    },
    "currentSensitiveState": {
      "// Current sensitive computed state": "..."
    }
  },
  "id": 7
}
```

**Note**: For create operations, `id`, `currentProps`, and `currentState` will be `null`. For delete operations, `nextProps` will be `null` (only `currentProps` and `currentState` are provided).

#### Response (No Changes)

```json
{
  "jsonrpc": "2.0",
  "result": {
    "noChanges": true
  },
  "id": 7
}
```

#### Response (Modified Props)

```json
{
  "jsonrpc": "2.0",
  "result": {
    "modifiedProps": {
      "// Modified configuration values": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Property normalized",
        "detail": "The value was normalized to a standard format",
        "propPath": ["nextProps", "example_property"]
      }
    ]
  },
  "id": 7
}
```

#### Response (Requires Replacement)

```json
{
  "jsonrpc": "2.0",
  "result": {
    "requiresReplacement": true
  },
  "id": 7
}
```

#### OpenRPC Schema

```json
{
  "name": "modifyPlan",
  "description": "Optional method to modify planned values or indicate replacement is required",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "id": {
            "type": ["string", "null"],
            "description": "Resource identifier (null for create operations)"
          },
          "planType": {
            "type": "string",
            "enum": ["create", "update", "delete"],
            "description": "Type of operation being planned"
          },
          "nextProps": {
            "type": "object",
            "description": "Proposed new configuration properties"
          },
          "currentProps": {
            "type": ["object", "null"],
            "description": "Current configuration properties (null for create)"
          },
          "currentState": {
            "type": ["object", "null"],
            "description": "Current computed state (null for create)"
          },
          "currentSensitiveState": {
            "type": ["object", "null"],
            "description": "Current sensitive computed state (null for create)"
          }
        },
        "required": ["planType", "nextProps"]
      }
    }
  ],
  "result": {
    "name": "modifyPlanResult",
    "schema": {
      "oneOf": [
        {
          "type": "object",
          "properties": {
            "noChanges": {
              "type": "boolean",
              "const": true
            }
          },
          "required": ["noChanges"]
        },
        {
          "type": "object",
          "properties": {
            "modifiedProps": {
              "type": "object",
              "description": "Modified configuration values"
            },
            "diagnostics": {
              "type": "array",
              "items": {
                "type": "object",
                "properties": {
                  "severity": {
                    "type": "string",
                    "enum": ["error", "warning"]
                  },
                  "summary": {
                    "type": "string"
                  },
                  "detail": {
                    "type": "string"
                  },
                  "propPath": {
                    "type": "array",
                    "items": {
                      "type": "string"
                    },
                    "description": "Path to the property this diagnostic relates to"
                  }
                },
                "required": ["severity", "summary", "detail"]
              }
            }
          }
        },
        {
          "type": "object",
          "properties": {
            "requiresReplacement": {
              "type": "boolean"
            }
          },
          "required": ["requiresReplacement"]
        }
      ]
    }
  },
  "errors": [
    {
      "code": -32601,
      "message": "Method not found",
      "description": "Returned when modifyPlan is not implemented"
    }
  ]
}
```

## Data Source Provider

Data sources perform read-only operations to retrieve information from external systems.

### read

**Direction**: Go → Deno

Reads data from an external source based on the provided configuration.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "read",
  "params": {
    "props": {
      "// Query parameters": "..."
    }
  },
  "id": 8
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "result": {
      "// Retrieved data": "..."
    },
    "sensitiveResult": {
      "// Sensitive retrieved data": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Data may be stale",
        "detail": "The retrieved data is from a cache and may be slightly out of date"
      }
    ]
  },
  "id": 8
}
```

**Fields:**

- `result` (required): The data retrieved from the external source
- `sensitiveResult` (optional): Sensitive data (marked as sensitive in Terraform, not displayed in logs or plan output)
- `diagnostics` (optional): Warnings or errors to display to the user

#### OpenRPC Schema

```json
{
  "name": "read",
  "description": "Reads data from an external source",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "props": {
            "type": "object",
            "description": "Configuration/query parameters for the data source"
          }
        },
        "required": ["props"]
      }
    }
  ],
  "result": {
    "name": "readResult",
    "schema": {
      "type": "object",
      "properties": {
        "result": {
          "type": "object",
          "description": "Retrieved data from the external source"
        },
        "sensitiveResult": {
          "type": "object",
          "description": "Sensitive retrieved data from the external source (marked as sensitive in Terraform)"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["result"]
    }
  }
}
```

## Ephemeral Resource Provider

Ephemeral resources represent temporary data that is made available during Terraform operations but not persisted in state.

### open

**Direction**: Go → Deno

Opens an ephemeral resource, optionally with automatic renewal.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "open",
  "params": {
    "props": {
      "// Configuration properties": "..."
    }
  },
  "id": 9
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "result": {
      "// Ephemeral data": "..."
    },
    "sensitiveResult": {
      "// Sensitive ephemeral data": "..."
    },
    "renewAt": 1735891200000,
    "privateData": {
      "// Internal data for renewal/close": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Credential expiry",
        "detail": "The ephemeral credential will expire in 1 hour"
      }
    ]
  },
  "id": 9
}
```

**Fields:**

- `result` (required): The ephemeral data to be made available
- `sensitiveResult` (optional): Sensitive ephemeral data (marked as sensitive in Terraform, not displayed in logs or plan output)
- `renewAt` (optional): Unix timestamp in seconds when renewal should occur
- `privateData` (optional): Private data passed back to renew/close methods (not exposed to Terraform)
- `diagnostics` (optional): Warnings or errors to display to the user

#### OpenRPC Schema

```json
{
  "name": "open",
  "description": "Opens an ephemeral resource",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "props": {
            "type": "object",
            "description": "Configuration properties for the ephemeral resource"
          }
        },
        "required": ["props"]
      }
    }
  ],
  "result": {
    "name": "openResult",
    "schema": {
      "type": "object",
      "properties": {
        "result": {
          "type": "object",
          "description": "The ephemeral data to be made available"
        },
        "sensitiveResult": {
          "type": "object",
          "description": "Sensitive ephemeral data (marked as sensitive in Terraform)"
        },
        "renewAt": {
          "type": "integer",
          "description": "Unix timestamp in seconds when renewal should occur"
        },
        "privateData": {
          "type": "object",
          "description": "Private data for internal use (passed to renew/close)"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["result"]
    }
  }
}
```

### renew (Optional)

**Direction**: Go → Deno

Renews an ephemeral resource before it expires. This method is optional.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "renew",
  "params": {
    "privateData": {
      "// Private data from open/previous renew": "..."
    }
  },
  "id": 10
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "renewAt": 1735894800000,
    "privateData": {
      "// Updated private data": "..."
    },
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Renewal rate limit",
        "detail": "Approaching rate limit for renewal operations"
      }
    ]
  },
  "id": 10
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "renew",
  "description": "Optional method to renew an ephemeral resource",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "privateData": {
            "type": "object",
            "description": "Private data from open or previous renew call"
          }
        },
        "required": ["privateData"]
      }
    }
  ],
  "result": {
    "name": "renewResult",
    "schema": {
      "type": "object",
      "properties": {
        "renewAt": {
          "type": "integer",
          "description": "Unix timestamp in seconds for next renewal"
        },
        "privateData": {
          "type": "object",
          "description": "Updated private data for next renewal or close"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      }
    }
  },
  "errors": [
    {
      "code": -32601,
      "message": "Method not found",
      "description": "Returned when renew is not implemented"
    }
  ]
}
```

### close (Optional)

**Direction**: Go → Deno

Closes an ephemeral resource and performs cleanup. This method is optional.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "close",
  "params": {
    "privateData": {
      "// Private data from open/renew": "..."
    }
  },
  "id": 11
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "done": true,
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Cleanup incomplete",
        "detail": "Some temporary resources could not be cleaned up automatically"
      }
    ]
  },
  "id": 11
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "close",
  "description": "Optional method to close an ephemeral resource and perform cleanup",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "privateData": {
            "type": "object",
            "description": "Private data from open or renew call"
          }
        },
        "required": ["privateData"]
      }
    }
  ],
  "result": {
    "name": "closeResult",
    "schema": {
      "type": "object",
      "properties": {
        "done": {
          "type": "boolean",
          "description": "Must be true to indicate successful closure"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["done"]
    }
  },
  "errors": [
    {
      "code": -32601,
      "message": "Method not found",
      "description": "Returned when close is not implemented"
    }
  ]
}
```

## Action Provider

Actions perform operations that don't manage state but may need to report progress.

### invoke

**Direction**: Go → Deno

Invokes an action operation.

#### Request

```json
{
  "jsonrpc": "2.0",
  "method": "invoke",
  "params": {
    "props": {
      "// Action parameters": "..."
    }
  },
  "id": 12
}
```

#### Response

```json
{
  "jsonrpc": "2.0",
  "result": {
    "done": true,
    "diagnostics": [
      {
        "severity": "warning",
        "summary": "Action partially completed",
        "detail": "Some operations completed with warnings"
      }
    ]
  },
  "id": 12
}
```

**Note**: The `diagnostics` field is optional and can be omitted if there are no warnings or errors to report.

#### OpenRPC Schema

```json
{
  "name": "invoke",
  "description": "Invokes an action operation",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "props": {
            "type": "object",
            "description": "Parameters for the action"
          }
        },
        "required": ["props"]
      }
    }
  ],
  "result": {
    "name": "invokeResult",
    "schema": {
      "type": "object",
      "properties": {
        "done": {
          "type": "boolean",
          "description": "Must be true to indicate successful completion"
        },
        "diagnostics": {
          "type": "array",
          "description": "Optional warnings or errors to display to the user",
          "items": {
            "type": "object",
            "properties": {
              "severity": {
                "type": "string",
                "enum": ["error", "warning"],
                "description": "Diagnostic severity level"
              },
              "summary": {
                "type": "string",
                "description": "Short description of the diagnostic"
              },
              "detail": {
                "type": "string",
                "description": "Additional context about the diagnostic"
              },
              "propPath": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "description": "Path to the property this diagnostic relates to"
              }
            },
            "required": ["severity", "summary", "detail"]
          }
        }
      },
      "required": ["done"]
    }
  }
}
```

### invokeProgress

**Direction**: Deno → Go

A notification sent from Deno to Go to report progress during action execution. This is the only method where the Deno process initiates communication.

#### Notification (No Response Expected)

```json
{
  "jsonrpc": "2.0",
  "method": "invokeProgress",
  "params": {
    "message": "Processing item 5 of 10..."
  }
}
```

#### OpenRPC Schema

```json
{
  "name": "invokeProgress",
  "description": "Reports progress during action execution (notification only, no response)",
  "params": [
    {
      "name": "params",
      "required": true,
      "schema": {
        "type": "object",
        "properties": {
          "message": {
            "type": "string",
            "description": "Progress message to display"
          }
        },
        "required": ["message"]
      }
    }
  ]
}
```

## Implementation Example

Here's a minimal example of implementing a resource provider from scratch in TypeScript/Deno:

```typescript
#!/usr/bin/env -S deno run --allow-read --allow-write

import { TextLineStream } from "jsr:@std/streams";

// Simple message ID counter
let messageId = 0;

// Parse incoming JSON-RPC messages
async function readMessages() {
  const decoder = new TextDecoderStream();
  const lineStream = Deno.stdin.readable
    .pipeThrough(decoder)
    .pipeThrough(new TextLineStream());

  for await (const line of lineStream) {
    if (!line.trim()) continue;

    const message = JSON.parse(line);
    await handleMessage(message);
  }
}

// Handle incoming requests
async function handleMessage(message: any) {
  if (!message.method) return; // Ignore responses

  let result: any;

  try {
    switch (message.method) {
      case "health":
        result = { ok: true };
        break;

      case "create":
        result = await createResource(message.params.props);
        break;

      case "read":
        result = await readResource(message.params.id, message.params.props);
        break;

      case "update":
        const state = await updateResource(
          message.params.id,
          message.params.nextProps,
          message.params.currentProps,
          message.params.currentState,
        );
        result = { state };
        break;

      case "delete":
        await deleteResource(
          message.params.id,
          message.params.props,
          message.params.state,
        );
        result = { done: true };
        break;

      case "shutdown":
        result = {};
        sendResponse(message.id, result);
        Deno.exit(0);
        return;

      default:
        sendError(message.id, -32601, "Method not found");
        return;
    }

    sendResponse(message.id, result);
  } catch (error) {
    sendError(message.id, -32000, error.message);
  }
}

// Send JSON-RPC response
function sendResponse(id: number, result: any) {
  const response = {
    jsonrpc: "2.0",
    result,
    id,
  };
  console.log(JSON.stringify(response));
}

// Send JSON-RPC error
function sendError(id: number, code: number, message: string) {
  const response = {
    jsonrpc: "2.0",
    error: { code, message },
    id,
  };
  console.log(JSON.stringify(response));
}

// Implement your resource lifecycle methods
async function createResource(props: any) {
  // Your creation logic here
  return {
    id: "generated-id",
    state: {/* computed values */},
  };
}

async function readResource(id: string, props: any) {
  // Your read logic here
  return {
    props: {/* refreshed props */},
    state: {/* refreshed state */},
  };
  // Or return { exists: false } if not found
}

async function updateResource(
  id: string,
  nextProps: any,
  currentProps: any,
  currentState: any,
) {
  // Your update logic here
  return {/* new computed state */};
}

async function deleteResource(id: string, props: any, state: any) {
  // Your deletion logic here
}

// Start the JSON-RPC server
readMessages();
```

## Error Handling

The JSON-RPC 2.0 specification defines standard error codes:

| Code             | Message          | Meaning                                           |
| ---------------- | ---------------- | ------------------------------------------------- |
| -32700           | Parse error      | Invalid JSON was received                         |
| -32600           | Invalid Request  | The JSON sent is not a valid Request object       |
| -32601           | Method not found | The method does not exist or is not available     |
| -32602           | Invalid params   | Invalid method parameter(s)                       |
| -32603           | Internal error   | Internal JSON-RPC error                           |
| -32000 to -32099 | Server error     | Reserved for implementation-defined server errors |

When an error occurs in your provider implementation, return an error response:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Failed to create resource: connection timeout"
  },
  "id": 3
}
```

## Debugging

Enable debug logging by setting the `TF_LOG` environment variable to `debug`:

```bash
TF_LOG=debug terraform plan
```

This will cause the Go provider to output all STDERR from the Deno child process as DEBUG logs.

Thus you may elect to also configure your Deno script to read this env var and output logs to STDERR.

**Never output anything but JSON-RPC messages to STDOUT!**

## Complete OpenRPC Document

A complete OpenRPC specification document is available that can be used with tools like [OpenRPC Playground](https://playground.open-rpc.org/):

**[View the OpenRPC Specification](https://raw.githubusercontent.com/brad-jones/terraform-provider-denobridge/refs/heads/master/docs/guides/json-rpc-spec.json)**

You can load this specification into the OpenRPC Playground to explore the API interactively:

1. Go to [OpenRPC Playground](https://playground.open-rpc.org/)
2. Click "File" → "Load From URL"
3. Enter: `https://raw.githubusercontent.com/brad-jones/terraform-provider-denobridge/refs/heads/master/docs/guides/json-rpc-spec.json`

## Further Reading

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [OpenRPC Specification](https://spec.open-rpc.org/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Deno Runtime Documentation](https://docs.deno.com/)
