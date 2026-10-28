# REST API through a local bridge

This example starts `my-api-bridge` on the machine running Terraform or OpenTofu, waits ten seconds with the time provider, then creates an object through [Mastercard/restapi](https://registry.terraform.io/providers/Mastercard/restapi/latest/docs). The bridge process closes when the operation ends.

Replace the command arguments, `/api/widgets` path, and JSON body with those supported by your bridge. The example assumes POST returns a JSON object with an `id`, and the API supports GET, PUT, and DELETE at `/api/widgets/{id}`. Once the subprocess provider is listed in the Terraform Registry, run from [`examples/restapi-bridge`](https://github.com/alexanderankin/terraform-provider-subprocess/tree/main/examples/restapi-bridge) with `tofu init && tofu apply`.

Terraform configures providers before resource creation, so the REST provider's `uri` is a static local address. The subprocess waits for a TCP listener on the bridge port. The dependency orders its **resource request** after the process and the ten second delay on initial creation. `time_sleep` does not wait again on an unchanged apply. The bridge must be reachable from the same machine running Terraform.

```hcl
--8<-- "examples/restapi-bridge/main.tf"
```
