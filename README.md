# subprocess provider

Run a local child process while OpenTofu or Terraform performs dependent operations. This is an early release: the process is started in an ephemeral resource's `Open` and killed in `Close`.

```hcl
terraform {
  required_providers {
    subprocess = {
      source  = "registry.terraform.io/alexanderankin/subprocess"
      version = "~> 0.1.0"
    }
  }
}

ephemeral "subprocess_process" "service" {
  command          = ["my-service", "--listen", "127.0.0.1:12345"]
  startup_delay_ms = 1000
}

resource "some_provider_resource" "example" {
  depends_on = [ephemeral.subprocess_process.service]
}
```

The process starts when OpenTofu needs the ephemeral resource, potentially once during plan and again during apply. `depends_on` orders dependent operations after startup. `startup_delay_ms` is a fixed delay, not a readiness check. The provider kills the direct child when OpenTofu closes the ephemeral resource; it does not manage grandchildren. This first version discards child stdout and stderr.

Run `./smoke_test.sh` to verify the lifecycle with OpenTofu.

See the [HTML documentation](https://alexanderankin.github.io/terraform-provider-subprocess/) for the [MySQL port forward](https://alexanderankin.github.io/terraform-provider-subprocess/examples/mysql-port-forward/) and [REST API bridge](https://alexanderankin.github.io/terraform-provider-subprocess/examples/restapi-bridge/) examples.
