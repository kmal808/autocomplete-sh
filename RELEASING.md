# Releasing

This fork publishes from `github.com/kmal808/autocomplete-sh`.

## Preflight

- `git status` is clean
- `go test ./...` passes locally
- `go build ./cmd/autocomplete` succeeds locally
- `README.md`, `USAGE.md`, `NOTICE`, and `docs/index.html` reflect the current release behavior
- `docs/install.sh` still points at `kmal808/autocomplete-sh`
- the GitHub repo has the right description, homepage, and topics from `REPO_METADATA.md`

## Release Steps

1. Update `version.go` if the release version string should change.
2. Commit the release-ready state to `main`.
3. Push `main` to `origin`.
4. Create and push a tag:

```bash
git tag v0.6.0
git push origin v0.6.0
```

5. Wait for `.github/workflows/release.yml` to publish release artifacts:
   - `autocomplete_darwin_arm64.tar.gz`
   - `autocomplete_darwin_amd64.tar.gz`
   - `autocomplete_linux_arm64.tar.gz`
   - `autocomplete_linux_amd64.tar.gz`
6. Verify the GitHub Release page contains all four artifacts.
7. Smoke-test the published installer:

```bash
curl -fsSL https://raw.githubusercontent.com/kmal808/autocomplete-sh/main/docs/install.sh | sh
acsh version
autocomplete doctor
```

## Post-release

- Update release notes with major changes, breakages, and migration notes
- If you want upstream visibility, open a short issue linking to the release after the release is live
