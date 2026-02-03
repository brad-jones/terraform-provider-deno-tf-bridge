resource "denobridge_resource" "quote_of_the_day" {
  # The path to the underlying deno script.
  # Remote HTTP URLs also supported! Any valid value that `deno run` will accept.
  path = "${path.module}/resource.ts"

  # The inputs required by the underlying deno script to read the data.
  props = {
    path = "quote.txt"
    content = "Minim cillum nisi reprehenderit enim mollit deserunt exercitation aliqua in mollit ex."
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
  # Computed state (aka: outputs) available after the resource has been
  # created can be read from the dynamic & untyped state attribute.
  last_updated = resource.denobridge_resource.quote_of_the_day.state.mtime
}
