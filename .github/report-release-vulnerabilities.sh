#!/usr/bin/env bash

set -euo pipefail

# download the release.yaml
gh release download --clobber --pattern release.yaml --output /tmp/release.yaml

# extract the images
readarray -t images < <(grep ghcr.io /tmp/release.yaml | sed -E 's/(image|value)://' | tr -d ' ' | sort -u)

# capture whether vulnerabilities exist
hasVulnerabilities=false

# iterate the images
true>/tmp/report.md
for image in "${images[@]}"; do
  echo "[INFO] Checking image ${image}"
  echo "## ${image}" >>/tmp/report.md

  # OS vulnerabilities
  echo "  [INFO] Checking for OS vulnerabilities"
  echo "### OS vulnerabilities" >>/tmp/report.md
  osVulns="$(trivy image --format json --ignore-unfixed --no-progress --pkg-types os --scanners vuln --skip-db-update --timeout 10m "${image}")"
  osVulnsFound=false
  while read -r id pkg severity vulnerableVersion fixedVersion; do
    if [ "${id}" == "" ]; then
      continue
    fi

    if [ "${osVulnsFound}" == "false" ]; then
      echo "| Vulnerability | Package | Severity | Version |" >>/tmp/report.md
      echo "| -- | -- | -- | -- |" >>/tmp/report.md
      osVulnsFound=true
      hasVulnerabilities=true
    fi

    severityLower="$(tr '[:upper:]' '[:lower:]' <<<"${severity}")"

    echo "    [INFO] Found ${id} in ${pkg} with severity ${severityLower}. Requires upgrade from ${vulnerableVersion} to ${fixedVersion}."
    echo "| ${id} | ${pkg} | ${severityLower} | ${vulnerableVersion} -> ${fixedVersion} |" >>/tmp/report.md
  done <<<"$(jq --raw-output '.Results[0].Vulnerabilities[] | [ .VulnerabilityID, .PkgName, .Severity, .InstalledVersion, .FixedVersion ] | @tsv' <<<"${osVulns}")"

  if [ "${osVulnsFound}" == "false" ]; then
    echo "    [INFO] No vulnerabilities found."
    echo "No vulnerabilities found." >>/tmp/report.md
  fi

  # Go vulnerabilities
  echo "  [INFO] Checking for Go vulnerabilities"
  echo "### Go vulnerabilities" >>/tmp/report.md
  entrypoint="$(crane config "${image}" | jq -r '.config.Entrypoint[0]')"
  crane export "${image}" - | tar -xf - -C /tmp "${entrypoint}"
  goVulns="$(govulncheck -format json -mode binary "/tmp${entrypoint}")"
  goVulnsFound=false
  while read -r id pkg vulnerableVersion fixedVersion; do
    if [ "${id}" == "" ]; then
      continue
    fi

    if [ "${goVulnsFound}" == "false" ]; then
      echo "| Vulnerability | Package | Version |" >>/tmp/report.md
      echo "| -- | -- | -- |" >>/tmp/report.md
      goVulnsFound=true
      hasVulnerabilities=true
    fi

    echo "    [INFO] Found ${id} in ${pkg}. Requires upgrade from ${vulnerableVersion} to ${fixedVersion}."
    echo "| ${id} | ${pkg} | ${vulnerableVersion} -> ${fixedVersion} |" >>/tmp/report.md
  done <<<"$(jq --raw-output 'select(.finding != null and .finding.fixed_version != null) | [ .finding.osv, .finding.trace[0].module, .finding.trace[0].version, .finding.fixed_version ] | @tsv' <<<"${goVulns}" | sort -u)"

  if [ "${goVulnsFound}" == "false" ]; then
    echo "    [INFO] No vulnerabilities found."
    echo "No vulnerabilities found." >>/tmp/report.md
  fi
done

# check if issue exists, if yes, update description, otherwise create one, or close it if vulnerabilities are gone
issues="$(gh issue list --label release-vulnerabilities --json number)"

if [ "$(jq length <<<"${issues}")" == "0" ]; then
  if [ "${hasVulnerabilities}" == "true" ]; then
    # create new issue
    echo "[INFO] Creating new issue"
    gh issue create --label release-vulnerabilities --title "Vulnerabilities found in latest release" --body-file /tmp/report.md
  fi
else
  issueNumber="$(jq '.[0].number' <<<"${issues}")"
  if [ "${hasVulnerabilities}" == "true" ]; then
    # update issue
    echo "[INFO] Updating existing issue ${issueNumber}"
    gh issue edit "${issueNumber}" --body-file /tmp/report.md
  else
    gh issue close --reason "No vulnerabilities found in the latest release"
  fi
fi
