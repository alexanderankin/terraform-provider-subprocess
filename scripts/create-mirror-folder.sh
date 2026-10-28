#!/usr/bin/env bash
if [[ "$0" != "$BASH_SOURCE" ]]; then echo "no sourcing">&2; return 1; fi;
set -eu -o pipefail
[[ ${DEBUG:-} ]] && set -x
full="$(readlink -f "$BASH_SOURCE")"; dir="${full%\/*}"; file="${full##*/}";

if [[ $# -gt 1 ]]; then
  echo "Usage: $0 MIRROR_FOLDER" >&2
  exit 2
fi

mirror_root_default=~/.local/tf-providers
mirror_root="${1:-${mirror_root_default}}"
mkdir -p "${mirror_root}"
echo "using mirror root '${mirror_root}'"

platform="$(go env GOOS)_$(go env GOARCH)"
provider_dir="${mirror_root}/registry.terraform.io/alexanderankin/subprocess/0.1.0/$platform"

mkdir -p "${provider_dir}"
repo_dir="$(cd "${dir}"; cd ..; pwd)"
go -C "${repo_dir}" build -o "${provider_dir}/terraform-provider-subprocess_v0.1.0" .
echo "Built provider in ${provider_dir}"
