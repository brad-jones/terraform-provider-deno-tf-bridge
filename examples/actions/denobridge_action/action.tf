action "denobridge_action" "launch_rocket" {
  # The path to the underlying deno script.
  # Remote HTTP URLs also supported! Any valid value that `deno serve` will accept.
  path = "${path.module}/action.ts"

  # The inputs required by the underlying deno script to invoke the action.
  props = {
    destination = "mars"
  }

  # Optionally set any runtime permissions that the deno script may require.
  permissions = {
    all = true # Maps to --allow-all (use with caution!)

    # Otherwise provide the exact permissions your script needs.
    # see: https://docs.deno.com/runtime/fundamentals/security/#permissions
    allow = ["read", "net=example.com:443"]
    deny = ["write"]
  }
}
