import {
  to = denobridge_resource.quote_of_the_day

  # The Import ID is a JSON encoded string
  id = jsonencode({
    # At a minimum you need to provide the path to the deno script
    path = "${path.module}/resource.ts"

    # Optionally you may need to set deno permissions
    permissions = {
      all = true
    }

    # Of course the id of the actual resource must be given too.
    id = "quote.txt"

    # And depending on the implementation of the resource script you may also
    # need to supply some of the props so that the resource can be uniquely identified.
    props = {
      foo = "bar"
    }
  })
}
