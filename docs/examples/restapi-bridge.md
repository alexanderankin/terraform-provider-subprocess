# REST API through a local bridge

This example starts `my-api-bridge` on the machine running Terraform or OpenTofu, waits for its TCP listener and a ten-second startup delay, then creates an object through [Mastercard/restapi](https://registry.terraform.io/providers/Mastercard/restapi/latest/docs). The bridge process closes when the operation ends.

Replace the command arguments, `/api/widgets` path, and JSON body with those supported by your bridge. The example assumes POST returns a JSON object with an `id`, and the API supports GET, PUT, and DELETE at `/api/widgets/{id}`. Once the subprocess provider is listed in the Terraform Registry, run from [`examples/restapi-bridge`](https://github.com/alexanderankin/terraform-provider-subprocess/tree/main/examples/restapi-bridge) with `tofu init && tofu apply`.

The process passes its local URI through `inputs` to its computed `outputs`. The REST provider references that output, so its configuration depends on the bridge opening and completing both wait conditions. The startup delay applies whenever the ephemeral process opens, including on a later apply. The bridge must be reachable from the same machine running Terraform.

```hcl
--8<-- "examples/restapi-bridge/main.tf"
```
