---
page_title: "subprocess_process Ephemeral Resource"
subcategory: ""
description: |-
  Start a local process and stop it after dependent operations.
---

# subprocess_process Ephemeral Resource

```hcl
ephemeral "subprocess_process" "service" {
  command          = ["my-service", "--listen", "127.0.0.1:12345"]
  startup_delay_ms = 1000
}
```

Use `depends_on = [ephemeral.subprocess_process.service]` on operations requiring the process.

## Arguments

- `command` (required): Executable and arguments. A shell is used only if explicitly included.
- `working_directory` (optional): Process working directory.
- `environment` (optional): Additional environment variables.
- `startup_delay_ms` (optional): Fixed wait after starting, from 0 to 300000 milliseconds.

## Attributes

- `pid`: Child process ID.
