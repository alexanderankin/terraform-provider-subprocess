---
page_title: "subprocess Provider"
subcategory: ""
description: |-
  Run a local child process during dependent OpenTofu or Terraform operations.
---

# subprocess Provider

The subprocess provider exposes an ephemeral process resource. It starts the configured command when opened and kills the direct child when closed. See the [process resource](ephemeral-resources/process.md).

## Examples

- [Create a MySQL user through an SSH port forward](examples/mysql-port-forward.md)
- [Use a local API bridge with Mastercard/restapi](examples/restapi-bridge.md)
