---
name: dependency-bump
description: Bump ytclipper's Go version, Go modules, GitHub Actions or the tools baked into the Docker image (yt-dlp, deno) safely. Use when Dependabot PRs pile up, a security advisory lands, or yt-dlp stops working with YouTube.
---

# Bumping dependencies

## Go and modules
1. Keep the Go version in sync across `go.mod` (`go` directive), the Dockerfile builder image and `GO_VERSION` in the CI workflows.
2. `go get -u ./... && go mod tidy`, then `go vet ./... && go test ./...`.
3. Read the changelog of anything with a major or minor jump (Echo, chromedp) before trusting green tests: the e2e suite is the only coverage for chromedp.
4. Close superseded Dependabot PRs with a comment pointing at your PR.

## yt-dlp and deno (Docker image)
- yt-dlp breaks when YouTube changes; the image installs the latest at build time, so a rebuild usually fixes extraction. If you pin it, pin to a release tag and say why in the Dockerfile.
- deno is pinned in the Dockerfile; bump it deliberately and check `deno --version` in the build log.
- These can't be verified in CI against real YouTube. Note that in the PR, and check the deployed instance after merge.

## GitHub Actions
- Dependabot also watches `github-actions`; bump major versions of `actions/*` and `docker/*` together and check the workflow run on the PR.
