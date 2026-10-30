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

version=$(cat "${dir}/../.release-please-manifest.json" | jq -r '."."')

platform="$(go env GOOS)_$(go env GOARCH)"

registries=(registry.terraform.io registry.opentofu.org)
for each_registry in "${registries[@]}"
do
  provider_dir="${mirror_root}/${each_registry}/alexanderankin/subprocess/${version}/$platform"

  mkdir -p "${provider_dir}"
  repo_dir="$(cd "${dir}"; cd ..; pwd)"
  go -C "${repo_dir}" build -o "${provider_dir}/terraform-provider-subprocess_v${version}" .
  echo "Built provider in ${provider_dir}"
done
