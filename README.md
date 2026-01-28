<div align="center">

### 🚀 Build Terraform Providers with TypeScript

_Bridge the infrastructure-as-code world with the TypeScript ecosystem_

[![Built with Deno](https://img.shields.io/badge/Built%20with-Deno-00ADD8?style=flat&logo=deno)](https://deno.com/)
[![Terraform](https://img.shields.io/badge/Terraform-844FBA?style=flat&logo=terraform&logoColor=white)](https://registry.terraform.io/providers/brad-jones/denobridge/latest)
[![OpenTofu](https://img.shields.io/badge/OpenTofu-FFDA18?style=flat&logo=opentofu&logoColor=black)](https://search.opentofu.org/provider/brad-jones/denobridge/latest)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](#)

---

</div>

## 🌟 Overview

**Deno Tofu Bridge** allows you to implement Terraform provider logic in **TypeScript** instead of Go.

Your TypeScript code runs as a Deno HTTP server, and the provider communicates with it via a well-defined HTTP API. This approach combines the **safety** 🔒 and **simplicity** ✨ of Deno with the **power** ⚡ of Terraform's infrastructure management.

Write provider logic in the language you know and love, with access to the entire npm ecosystem!

## ✨ Features

- 🔄 **Full Resource Lifecycle** - Implement complete CRUD operations for managed resources
- 📊 **Data Sources** - Fetch external data or perform computations
- ⏱️ **Ephemeral Resources** - Create short-lived resources for temporary credentials or tokens
- 🎯 **Actions** - Execute operations like notifications, validations, or external workflows
- 🛡️ **Type Safety** - Write provider logic in TypeScript with full type checking
- 🔐 **Deno Permissions** - Fine-grained security control over what your code can access
- 🌐 **HTTP-Based Protocol** - Simple, well-documented API contract between Go and TypeScript
- ⚡ **Fast Development** - Rapid prototyping without Go expertise
- 📦 **npm Ecosystem** - Leverage TypeScript's rich package ecosystem

## Quick Start

### Installation

Add the provider to your Terraform configuration:

```hcl
terraform {
  required_providers {
    denobridge = {
      source = "registry.terraform.io/brad-jones/denobridge"
    }
  }
}

provider "denobridge" {}
```

### Provider Configuration

The provider automatically manages the Deno binary, but you can customize this behavior:

#### Default (Automatic Download)

By default, the provider downloads the latest stable Deno release:

```hcl
provider "denobridge" {
  # No configuration needed - downloads latest Deno automatically
}
```

The provider caches up to 3 Deno versions in your system's temp directory to avoid repeated downloads.

#### Specify Deno Version

To use a specific Deno version, set `deno_version`:

```hcl
provider "denobridge" {
  deno_version = "v2.1.4"  # Use a specific version
}
```

Version format examples:

- `"latest"` - Latest stable GA release (default)
- `"v2.1.4"` - Specific stable version
- `"v2.0.0-rc.1"` - Pre-release version

The provider fetches available versions from the [Deno releases](https://github.com/denoland/deno/releases) on GitHub.

#### Use Custom Deno Binary

To use your own Deno installation, specify the binary path:

```hcl
provider "denobridge" {
  deno_binary_path = "/usr/local/bin/deno"  # Unix/macOS
  # or
  # deno_binary_path = "C:\\Program Files\\deno\\deno.exe"  # Windows
}
```

When `deno_binary_path` is set, the provider skips automatic downloading and uses your specified binary.

**Note**: You can also set the `GITHUB_TOKEN` environment variable to authenticate GitHub API requests, which helps avoid rate limiting when downloading Deno versions.

### Example: File Resource

Create a TypeScript file that manages a text file:

**providers/resource_file.ts**

```typescript
import { Hono } from "hono";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/create", async (c) => {
  const body = await c.req.json();
  const { path, content } = body.props;

  await Deno.writeTextFile(path, content);

  return c.json({
    id: path,
    state: {
      mtime: (await Deno.stat(path)).mtime!.getTime(),
    },
  });
});

app.post("/read", async (c) => {
  const body = await c.req.json();
  const { id } = body;

  try {
    const content = await Deno.readTextFile(id);
    return c.json({
      props: { path: id, content },
      state: {
        mtime: (await Deno.stat(id)).mtime!.getTime(),
      },
    });
  } catch (e) {
    if (e instanceof Deno.errors.NotFound) {
      return c.json({ exists: false });
    }
    throw e;
  }
});

app.post("/update", async (c) => {
  const body = await c.req.json();
  const { id } = body;
  const { content: nextContent } = body.nextProps;

  await Deno.writeTextFile(id, nextContent);

  return c.json({
    state: {
      mtime: (await Deno.stat(id)).mtime!.getTime(),
    },
  });
});

app.post("/delete", async (c) => {
  const body = await c.req.json();
  const { id } = body;

  await Deno.remove(id);
  return c.body(null, 204);
});

export default app satisfies Deno.ServeDefaultExport;
```

**main.tf**

```hcl
resource "denobridge_resource" "example_file" {
  path = "${path.module}/providers/resource_file.ts"

  permissions = {
    all = true
  }

  props = {
    path    = "${path.module}/output.txt"
    content = "Hello from Deno!"
  }
}
```

### Example: DNS Data Source

Query DNS records using Deno's built-in DNS resolver:

**providers/datasource_dns_record.ts**

```typescript
import { Hono } from "hono";

const app = new Hono();

app.get("/health", (c) => {
  return c.body(null, 204);
});

app.post("/read", async (c) => {
  const body = await c.req.json();
  const { query, recordType } = body.props;

  const result = await Deno.resolveDns(query, recordType, {
    nameServer: { ipAddr: "1.1.1.1", port: 53 },
  });

  return c.json(result);
});

export default app satisfies Deno.ServeDefaultExport;
```

**main.tf**

```hcl
data "denobridge_datasource" "mx_record" {
  path = "${path.module}/providers/datasource_dns_record.ts"

  permissions = {
    all = true
  }

  props = {
    query      = "example.com."
    recordType = "MX"
  }
}

output "mail_server" {
  value = data.denobridge_datasource.mx_record.result[0].exchange
}
```

## Resource Types

### `denobridge_resource`

Full CRUD lifecycle management for Terraform managed resources.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /create` - Create a new resource instance
- `POST /read` - Read the current state
- `POST /update` - Update an existing resource
- `POST /delete` - Delete a resource

**Optional Endpoints:**

- `POST /modify-plan` - Modify Terraform plans

**Configuration:**

```hcl
resource "denobridge_resource" "example" {
  path = "${path.module}/providers/my_resource.ts"

  permissions = {
    allow = [
      "read=/tmp",
      "write=/tmp",
      "net=api.example.com",
    ]
  }

  props = {
    # Your resource-specific properties
  }
}
```

### `denobridge_datasource`

Read-only data fetching for Terraform data sources.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /read` - Fetch data

**Configuration:**

```hcl
data "denobridge_datasource" "example" {
  path = "${path.module}/providers/my_datasource.ts"

  permissions = {
    allow = ["net=api.example.com"]
  }

  props = {
    # Your datasource-specific properties
  }
}
```

### `denobridge_ephemeral_resource`

Short-lived resources that exist only during Terraform operations.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /open` - Open/create the ephemeral resource

**Optional Endpoints:**

- `POST /renew` - Renew expiring resources
- `POST /close` - Clean up when no longer needed

**Configuration:**

```hcl
ephemeral "denobridge_ephemeral_resource" "example" {
  path = "${path.module}/providers/my_ephemeral.ts"

  permissions = {
    allow = ["net=auth.example.com"]
  }

  props = {
    # Your ephemeral resource properties
  }
}
```

### `denobridge_action`

Operations that don't manage resources but perform actions.

**Required Endpoints:**

- `GET /health` - Health check
- `POST /invoke` - Execute the action (with streaming progress)

**Configuration:**

```hcl
resource "denobridge_action" "example" {
  path = "${path.module}/providers/my_action.ts"

  permissions = {
    allow = ["net=webhook.example.com"]
  }

  props = {
    # Your action-specific properties
  }
}
```

## Deno Permissions

The provider supports Deno's [security and permissions model](https://docs.deno.com/runtime/fundamentals/security/#permissions). You can grant all permissions or specify individual allow/deny rules that map directly to Deno CLI flags.

### Basic Usage

```hcl
permissions = {
  all = true  # Maps to --allow-all (use with caution!)
}
```

### Fine-Grained Control

Grant specific permissions using the `allow` list:

```hcl
permissions = {
  allow = [
    "read",                    # --allow-read (all paths)
    "write=/tmp",             # --allow-write=/tmp
    "net=example.com:443",    # --allow-net=example.com:443
    "env=HOME,USER",          # --allow-env=HOME,USER
    "run=curl,whoami",        # --allow-run=curl,whoami
    "sys=hostname,osRelease", # --allow-sys=hostname,osRelease
    "ffi=/path/to/lib.so",    # --allow-ffi=/path/to/lib.so
    "hrtime",                 # --allow-hrtime
  ]
}
```

### Deny Specific Permissions

Deny takes precedence over allow:

```hcl
permissions = {
  allow = ["net"]              # Allow all network access
  deny  = ["net=evil.com"]     # Except evil.com
}
```

### Common Permission Types

- **`read`** - File system read access (e.g., `read`, `read=/tmp,/etc`)
- **`write`** - File system write access (e.g., `write=/tmp`)
- **`net`** - Network access (e.g., `net`, `net=example.com,api.example.com:443`)
- **`env`** - Environment variables (e.g., `env`, `env=HOME,USER`)
- **`run`** - Subprocess execution (e.g., `run=curl,whoami`)
- **`sys`** - System information (e.g., `sys=hostname,osRelease`)
- **`ffi`** - Foreign function interface (e.g., `ffi=/path/to/lib.so`)
- **`hrtime`** - High-resolution time measurement
- **`import`** - Dynamic imports from web (e.g., `import=example.com`)

See [Deno's permission documentation](https://docs.deno.com/runtime/fundamentals/security/#permissions) for complete details.

## API Documentation

Detailed OpenAPI specifications for each resource type are available in the [docs](docs/) directory:

- [denobridge_resource.yaml](docs/denobridge_resource.yaml) - Full resource lifecycle
- [denobridge_datasource.yaml](docs/denobridge_datasource.yaml) - Data sources
- [denobridge_ephemeral_resource.yaml](docs/denobridge_ephemeral_resource.yaml) - Ephemeral resources
- [denobridge_action.yaml](docs/denobridge_action.yaml) - Actions

See [docs/README.md](docs/README.md) for comprehensive API documentation and patterns.

## Development

### Prerequisites

- Go 1.25.5 or later
- [Deno](https://deno.com/) for running examples
- [Task](https://taskfile.dev/) for running build tasks

### Building

```bash
# Build the provider
task build

# Run tests
task test

# Run example
task run:example
```

### Project Structure

```
.
├── docs/                   # API specifications and documentation
├── example/                # Example Terraform configurations
│   └── providers/          # Example TypeScript implementations
├── internal/
│   └── provider/           # Provider implementation
├── bin/                    # Built binaries
├── main.go                 # Provider entry point
└── go.mod                  # Go dependencies
```

## How It Works

1. **Provider Initialization**: When Terraform/OpenTofu calls the provider, it reads your TypeScript file path and permissions configuration.

2. **Deno Server Start**: The provider automatically downloads (if needed) and starts Deno, running your TypeScript file as an HTTP server on a random port.

3. **Health Check**: The provider polls the `/health` endpoint until your server is ready.

4. **Lifecycle Management**: The provider calls appropriate endpoints (`/create`, `/read`, `/update`, `/delete`, etc.) based on Terraform operations.

5. **JSON Communication**: All data is exchanged via JSON, making it easy to work with in TypeScript.

6. **Cleanup**: When Terraform completes, the provider shuts down the Deno server.

## Use Cases

- **Rapid Prototyping**: Quickly build provider logic without Go expertise
- **API Integration**: Leverage TypeScript's rich ecosystem for API clients
- **Custom Resources**: Manage resources not supported by existing providers
- **Data Transformation**: Use TypeScript for complex data processing
- **Testing**: Write and test provider logic with familiar TypeScript tooling
- **Learning**: Understand Terraform provider concepts without learning Go

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Related Projects

- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [Deno](https://github.com/denoland/deno)
- [Hono](https://hono.dev/) - Recommended web framework for building endpoints

## Acknowledgments

Built with the Terraform Plugin Framework and powered by Deno's secure runtime.
