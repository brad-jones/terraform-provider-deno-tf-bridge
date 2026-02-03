action "denobridge_action" "launch_rocket" {
  # The path to the underlying deno script.
  # Remote HTTP URLs also supported! Any valid value that `deno run` will accept.
  path = "${path.module}/action.ts"

  # The inputs required by the underlying deno script to invoke the action.
  props = {
    destination = "mars"
  }

  # Optionally provide a path to a deno config file.
  #
  # If none is given, the denobridge provider will attempt to locate the closest
  # config file relative to the script path. This is to ensure that things like
  # import maps work as expected.
  #
  # If you wish to opt out of this automatic config discovery, supply the path "/dev/null".
  config_file = "/path/to/deno.json"

  # Optionally set any runtime permissions that the deno script may require.
  permissions = {
    all = true # Maps to --allow-all (use with caution!)

    # Otherwise provide the exact permissions your script needs.
    # see: https://docs.deno.com/runtime/fundamentals/security/#permissions
    allow = ["read", "net=example.com:443"]
    deny = ["write"]
  }
}
