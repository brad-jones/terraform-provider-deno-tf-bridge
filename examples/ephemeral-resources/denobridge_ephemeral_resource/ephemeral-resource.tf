ephemeral "denobridge_ephemeral_resource" "uuid" {
  # The path to the underlying deno script.
  # Remote HTTP URLs also supported! Any valid value that `deno run` will accept.
  path = "${path.module}/ephemeral-resource.ts"

  # The inputs required by the underlying deno script to read the data.
  props = {
    type = "v4"
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

resource "foo" "bar" {
  # The results are again untyped and dynamic based on the deno script returned
  # In this example we would receive a new UUID generated with crypto.randomUUID()
  id = ephemeral.denobridge_ephemeral_resource.uuid.result
}
