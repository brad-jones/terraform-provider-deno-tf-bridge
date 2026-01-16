# Deno Permissions

The provider supports Deno's [security and permissions model](https://docs.deno.com/runtime/fundamentals/security/#permissions).
You can grant all permissions or specify individual allow/deny rules that map directly to Deno CLI flags.

## Basic Usage

```hcl
permissions = {
  all = true  # Maps to --allow-all (use with caution!)
}
```

## Fine-Grained Control

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

## Deny Specific Permissions

Deny takes precedence over allow:

```hcl
permissions = {
  allow = ["net"]              # Allow all network access
  deny  = ["net=evil.com"]     # Except evil.com
}
```

## Common Permission Types

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
