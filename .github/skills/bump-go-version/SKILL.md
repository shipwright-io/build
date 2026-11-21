---
name: bump-go-version
description: Bump the Go version used by this repository. Use when asked to update, upgrade, or bump Go (for example from 1.26 to 1.27) in the GitHub Actions workflows and other places that pin a Go version.
---

# Bump the Go version

The Go version is pinned in several places. When bumping it, update **all** of the locations listed below in a single change so that CI, releases, and local builds stay consistent.

Use the minor version (for example `1.27`) as input. In GitHub Actions the version is written as `<minor>.x` (for example `'1.27.x'`) so that
`actions/setup-go` picks the latest patch release.

## 1. GitHub Actions workflows using `actions/setup-go`

Update the `go-version` input of every `actions/setup-go` step to the new
`<minor>.x` value inside the `.github/workflows` directory.

Do **not** change the second `setup-go` step ("Install Go version of latest release") in `.github/workflows/report-release-vulnerabilities.yaml`. It uses `"${{ steps.download-latest-release.outputs.go-version }}.x"` to install the Go version of the latest release and must stay dynamic.

To find all occurrences (and catch newly added workflows), run:

```sh
grep -rn -A2 'actions/setup-go' .github/workflows
```

## 2. Mirrored `golang` images in `.github/workflows/mirror-images.yaml`

The `mirror-images.yaml` workflow mirrors `library/golang:<minor>` images to `ghcr.io/shipwright-io/shipwright-samples`. Make sure the list contains the new minor version, adding missing entries in ascending order next to the existing `library/golang` lines. Keep the older entries unless asked to remove them.

## 3. `HACK.md`

Update the Go version in the development notes of `HACK.md` to the new minor version (without `.x`).

## 4. ko sample Builds

The ko sample Builds build this repository, so their `go-version` parameter value must be able to compile our code. Update the `go-version` param value to the new minor version (without `.x`, it is used as a tag of the `golang` image) in:

- `samples/v1alpha1/build/build_ko_cr.yaml`
- `samples/v1beta1/build/build_ko_cr.yaml`

## 5. ko build strategy default and its documentation

Keep the default of the ko build strategy's `go-version` parameter in sync with the new minor version (without `.x`):

- `default` of the `go-version` parameter in `samples/v1alpha1/buildstrategy/ko/buildstrategy_ko_cr.yaml`
- `default` of the `go-version` parameter in `samples/v1beta1/buildstrategy/ko/buildstrategy_ko_cr.yaml`
- Default column of the `go-version` row in the ko parameter table of `docs/buildstrategies.md`

## Do NOT change `go.mod`

Do **not** change the `go` directive in `go.mod`. It declares the minimum Go version required by our own code and the dependencies we use, not the compiler version we build with. It only changes when our code or a dependency requires a newer Go version.
