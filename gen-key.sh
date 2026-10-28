#!/usr/bin/env bash
if [[ "$0" != "$BASH_SOURCE" ]]; then echo "no sourcing">&2; return 1; fi;
set -eu -o pipefail
[[ ${DEBUG:-} ]] && set -x
full="$(readlink -f "$BASH_SOURCE")"; dir="${full%\/*}"; file="${full##*/}";

project=$(basename "${dir}")

(
  set -euo pipefail
  umask 077

  KEY_UID="${project} Publisher <publisher@${project}.com>"

  KEY_DIR="$(mktemp -d "openpgp.XXXXXX")"
  trap 'rm -rf ${KEY_DIR}' EXIT

  export GNUPGHOME="${KEY_DIR}/gnupg"
  mkdir -m 700 "${GNUPGHOME}"

  gpg --batch --pinentry-mode loopback --passphrase '' --quick-generate-key "$KEY_UID" rsa4096 default 0

  gpg --batch --armor --export "${KEY_UID}" > "${KEY_DIR}/public.asc"
  gpg --batch --armor --export-secret-keys "${KEY_UID}" > "${KEY_DIR}/private.asc"

  mkdir -p "${dir}"/.idea/
  cp -v "${KEY_DIR}/public.asc" "${KEY_DIR}/private.asc" "${dir}"/.idea/

  echo "Keys saved in: ${KEY_DIR}"
)
