resource "denobridge_resource" "quote_of_the_day" {
  # The path to the underlying deno script.
  # Remote HTTP URLs also supported! Any valid value that `deno serve` will accept.
  path = "${path.module}/resource.ts"

  # The inputs required by the underlying deno script to read the data.
  props = {
    path = "quote.txt"
    content = "Minim cillum nisi reprehenderit enim mollit deserunt exercitation aliqua in mollit ex."
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

resource "foo" "bar" {
  # Computed state (aka: outputs) available after the resource has been
  # created can be read from the dynamic & untyped state attribute.
  last_updated = resource.denobridge_resource.quote_of_the_day.state.mtime
}
