#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "$0")" && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

go -C "$repo_dir" build -o "$test_dir/terraform-provider-subprocess" .

cat > "$test_dir/main.tf" <<'EOF'
terraform {
  required_providers {
    subprocess = { source = "alexanderankin/subprocess" }
  }
}

ephemeral "subprocess_process" "service" {
  command           = ["sh", "-c", "echo $$ > process.pid; exec sleep 30"]
  working_directory = path.module
  startup_delay_ms  = 100
}

resource "terraform_data" "check" {
  depends_on = [ephemeral.subprocess_process.service]

  provisioner "local-exec" {
    working_dir = path.module
    command     = "kill -0 $(cat process.pid)"
  }
}
EOF

cat > "$test_dir/tofurc" <<EOF
provider_installation {
  dev_overrides {
    "alexanderankin/subprocess" = "$test_dir"
  }
  direct {}
}
EOF

TF_CLI_CONFIG_FILE="$test_dir/tofurc" tofu -chdir="$test_dir" apply -auto-approve -no-color

pid="$(cat "$test_dir/process.pid")"
if kill -0 "$pid" 2>/dev/null; then
  echo "child process $pid is still running after OpenTofu exited" >&2
  exit 1
fi
echo "Smoke test passed: child ran during apply and exited afterward"
