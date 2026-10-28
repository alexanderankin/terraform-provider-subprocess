#!/usr/bin/env bash
if [[ "$0" != "$BASH_SOURCE" ]]; then echo "no sourcing">&2; return 1; fi;
set -eu -o pipefail
[[ ${DEBUG:-} ]] && set -x
full="$(readlink -f "$BASH_SOURCE")"; dir="${full%\/*}"; file="${full##*/}";

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: ${0} CONFIG_FILE [MIRROR_FOLDER]" >&2
  exit 2
fi

config_file="${1}"
mirror_root_default=~/.local/tf-providers
mirror_root="${2:-${mirror_root_default}}"
mkdir -p "${mirror_root}"
echo "using mirror root '${mirror_root}'"

mkdir -p "$(dirname "${config_file}")"
cat > "${config_file}" <<EOF
plugin_cache_dir = "$HOME/.terraform.d/plugin-cache"
provider_installation {
  filesystem_mirror {
    path    = "${mirror_root}"
    include = ["registry.terraform.io/alexanderankin/subprocess"]
  }
  direct {
    exclude = ["registry.terraform.io/alexanderankin/subprocess"]
  }
}
EOF

echo "Wrote provider mirror config to '${config_file}'"
echo "export TF_CLI_CONFIG_FILE='${config_file}'"
