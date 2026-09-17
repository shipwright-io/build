#!/usr/bin/env bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
scenario="${1:-valid}"
test_root="$(mktemp -d)"
trap 'rm -rf "${test_root}"' EXIT

fake_bin="${test_root}/bin"
target_dir="${test_root}/home/bin"
mkdir -p "${fake_bin}" "${target_dir}"

cat >"${fake_bin}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >>"${CURL_LOG}"
if [[ "$*" == *"releases/tags/v1.30.2"* ]]; then
  printf 'release metadata\n'
  exit 0
fi

if [[ "$*" == *".sha1"* ]]; then
  if [[ "${SCENARIO}" == "mismatch" ]]; then
    printf '%040d  spruce-%s-amd64\n' 0 "${SCENARIO}"
  else
    printf '%s  spruce-%s-amd64\n' "$(printf spruce | /usr/bin/sha1sum | awk '{print $1}')" "${SCENARIO}"
  fi
  exit 0
fi

for ((i = 1; i <= $#; i++)); do
  if [[ "${!i}" == "--output" ]]; then
    next=$((i + 1))
    printf spruce >"${!next}"
    exit 0
  fi
done

exit 1
EOF

cat >"${fake_bin}/jq" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

query="${*: -1}"
if [[ "${query}" == *"sha1\") | not"* ]]; then
  printf 'https://example.test/spruce-%s-amd64\n' "${SCENARIO}"
elif [[ "${SCENARIO}" == "missing-checksum" ]]; then
  printf 'null\n'
else
  printf 'https://example.test/spruce-%s-amd64.sha1\n' "${SCENARIO}"
fi
EOF

cat >"${fake_bin}/uname" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" == "-m" ]]; then
  printf 'x86_64\n'
elif [[ "${SCENARIO}" == "darwin" ]]; then
  printf 'Darwin\n'
else
  printf 'Linux\n'
fi
EOF

cat >"${fake_bin}/sha1sum" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf 'sha1sum %s\n' "$*" >>"${CHECKSUM_LOG}"
exec /usr/bin/sha1sum "$@"
EOF

cat >"${fake_bin}/shasum" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf 'shasum %s\n' "$*" >>"${CHECKSUM_LOG}"
exec /usr/bin/sha1sum --check
EOF

chmod +x "${fake_bin}/curl" "${fake_bin}/jq" "${fake_bin}/uname" "${fake_bin}/sha1sum" "${fake_bin}/shasum"

PATH="${fake_bin}:${target_dir}:${PATH}" \
HOME="${test_root}/home" \
CURL_LOG="${test_root}/curl.log" \
CHECKSUM_LOG="${test_root}/checksum.log" \
SCENARIO="${scenario}" \
"${repo_root}/hack/install-spruce.sh" >"${test_root}/installer.log" 2>&1 && installer_succeeded=true || installer_succeeded=false

if [[ "${scenario}" == "mismatch" || "${scenario}" == "missing-checksum" ]]; then
  test "${installer_succeeded}" = false
  test ! -e "${target_dir}/spruce"
else
  test "${installer_succeeded}" = true
  test -x "${target_dir}/spruce"
fi

if [[ "${scenario}" == "missing-checksum" ]]; then
  grep -q 'Unable to locate checksum' "${test_root}/installer.log"
else
  grep -q -- '--check' "${test_root}/checksum.log"
fi
if [[ "${scenario}" == "darwin" ]]; then
  grep -q '^shasum ' "${test_root}/checksum.log"
elif [[ "${scenario}" != "missing-checksum" ]]; then
  grep -q '^sha1sum ' "${test_root}/checksum.log"
fi
if [[ "${scenario}" == "missing-checksum" ]]; then
  grep -q '^--fail --silent --location https://api.github.com/repos/' "${test_root}/curl.log"
else
  grep -q "spruce-${scenario}-amd64.sha1" "${test_root}/curl.log"
fi
