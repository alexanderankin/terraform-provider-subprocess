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
  command = ["my-service", "--listen", "127.0.0.1:12345"]
  wait_for_startup_delay_ms = 1000
  # wait_timeout_ms = 10000
}

resource "some_provider_resource" "example" {
  depends_on = [ephemeral.subprocess_process.service]
}
```

The process starts when OpenTofu needs the ephemeral resource, potentially once during plan and again during apply.
`depends_on` orders dependent operations after startup. `wait_for_startup_delay_ms` waits a fixed duration.
`wait_for_output_regex` matches complete newline-terminated lines from stdout or stderr, and each
`wait_for_listening_port` block checks a local TCP port. All configured conditions must succeed. `wait_timeout_ms`
optionally limits the whole wait; with no timeout, waiting ends on readiness, process exit, or cancellation. The
provider kills the direct child when OpenTofu closes the ephemeral resource; it does not manage grandchildren. Child
output is drained and never sent to provider RPC stdout.
If the child exits after `Open` but before `Close`, the provider records its PID and exit status in debug logs. Set
`TF_LOG=DEBUG` or `TF_LOG_PROVIDER_SUBPROCESS=DEBUG` to see that event.

Run `./smoke_test.sh` to verify the lifecycle with OpenTofu.

See the [HTML documentation](https://alexanderankin.github.io/terraform-provider-subprocess/) for
the [MySQL port forward](https://alexanderankin.github.io/terraform-provider-subprocess/examples/mysql-port-forward/)
and [REST API bridge](https://alexanderankin.github.io/terraform-provider-subprocess/examples/restapi-bridge/) examples.
