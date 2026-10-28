---
page_title: "subprocess_process Ephemeral Resource"
subcategory: ""
description: |-
  Start a local process and stop it after dependent operations.
---

# subprocess_process Ephemeral Resource

```hcl
ephemeral "subprocess_process" "service" {
  command                   = ["my-service", "--listen", "127.0.0.1:12345"]
  wait_for_startup_delay_ms = 1000
}
```

Use `depends_on = [ephemeral.subprocess_process.service]` on operations requiring the process.

## Arguments

- `command` (required): Executable and arguments. A shell is used only if explicitly included.
- `working_directory` (optional): Process working directory.
- `environment` (optional): Additional environment variables.
- `wait_for_startup_delay_ms` (optional): Fixed wait after starting, in non-negative milliseconds.
- `wait_for_output_regex` (optional): Go regular expression matched against complete lines from either stdout or stderr. Output must end in a newline to match.
- `wait_for_listening_port` (repeatable block): Wait for a connection to `127.0.0.1` at `port` (1–65535). `protocol` must be `"tcp"`; UDP cannot reliably confirm readiness. Every configured port must become reachable.
- `wait_timeout_ms` (optional): Maximum milliseconds for the entire wait after process start. Unset by default. Zero causes an immediate timeout when a wait condition exists.

All configured conditions must succeed. With no wait conditions, `Open` returns after starting the process. If the child exits before `Open` completes, `Open` fails and reports its exit. An exit after `Open` and before `Close` is logged with the PID and exit status at debug level; it cannot retroactively fail `Open`. Set `TF_LOG=DEBUG` or `TF_LOG_PROVIDER_SUBPROCESS=DEBUG` to see it. Child output continues to be drained after readiness and is never written to provider RPC stdout.

## Attributes

- `pid`: Child process ID.
