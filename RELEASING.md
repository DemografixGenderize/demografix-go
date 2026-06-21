# Releasing

This package is a Go module. There is no registry to publish to. The Go module
proxy (`proxy.golang.org`) and the package site (`pkg.go.dev`) fetch the code
directly from the git tag you push. A release is a pushed `vX.Y.Z` tag, nothing
more.

## One-time setup

- The repository must be **public**. `proxy.golang.org` and `pkg.go.dev` only
  fetch public repositories. A private repository will not resolve for
  consumers and will not appear on `pkg.go.dev`.
- The module path must match the repository URL. It does:
  `module github.com/DemografixGenderize/demografix-go` in `go.mod` maps to
  `https://github.com/DemografixGenderize/demografix-go`.

No registry account is required. No publish credentials are required. The
release workflow needs no secrets: it runs the test suite as a final gate and
creates a GitHub Release using the built-in `GITHUB_TOKEN`, which Actions
provides automatically.

## Cutting a release

Go modules carry no manifest version field. The version lives entirely in the
git tag, so there is nothing to bump in `go.mod`.

1. Make sure `main` is green and holds the code you want to ship.
2. Pick the next semver version, for example `v0.1.0`.
3. Tag the commit and push the tag:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

The `Release` workflow triggers on the `v*.*.*` tag. It runs `go test ./...` as
a final gate and, on success, creates a GitHub Release with auto-generated
notes.

## After the tag

The proxy picks up the new version on first request. Consumers install it with:

```sh
go get github.com/DemografixGenderize/demografix-go@v0.1.0
```

To confirm the proxy and the docs site have the version, query the proxy
directly:

```sh
curl https://proxy.golang.org/github.com/!demografix!genderize/demografix-go/@v/list
```

The `pkg.go.dev` page populates the first time someone requests the new version.
