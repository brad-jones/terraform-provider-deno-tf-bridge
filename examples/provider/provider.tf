provider "denobridge" {
  # Optionally set either a path to an existing deno binary to use or the version to download,
  # otherwise the latest GA version of deno will be downloaded from https://github.com/denoland/deno/releases
  deno_binary_path = "/path/to/deno"
  deno_version = "v1.2.3"
}
