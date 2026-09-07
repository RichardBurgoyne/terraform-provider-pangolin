# Terraform Provider for Pangolin

This is a Terraform provider for Pangolin, enabling infrastructure-as-code management of Pangolin resources through Terraform.

## Usage

To use this provider, add the following to your Terraform configuration:

```hcl
terraform {
  required_providers {
    pangolin = {
      source = "RichardBurgoyne/pangolin"
    }
  }
}

provider "pangolin" {
  # Configuration options here
}
```

## Documentation

For detailed documentation on available resources and data sources, see the [docs](./docs/) directory.

## Contributing

Contributions are welcome! To build and test the provider locally, run:

```bash
make build    # Build the provider
make test     # Run tests
make lint     # Run linter
make fmt      # Format code
```

See the [GNUmakefile](./GNUmakefile) for all available targets.

## Releasing

1. Merge everything intended for the release into `main`.
2. `git tag vX.Y.Z && git push origin vX.Y.Z`
3. The Release workflow builds, signs, and publishes to GitHub Releases.
   `test:` commits are excluded from the generated changelog.

First-time setup (once, by the repo owner): generate a dedicated GPG key
for this repo, add its armored private key and passphrase as the
`GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` repo secrets, and upload the public
key to the Terraform Registry publisher settings so the registry can
verify signed releases.

## Known limitations

- **Internal (password-based) users are not supported.** The Pangolin
  integration API itself rejects org user creation with `type = "internal"`
  ("Internal users are not supported yet"), so `pangolin_user` only supports
  OIDC-backed users. This isn't a client limitation to fast-follow on; it
  will be revisited if/when the upstream API adds support.
- **Removing a role from a `pangolin_user`** (a shrinking `role_ids`) calls a
  role-removal route that is part of Pangolin's commercial integration API,
  not the AGPL-licensed community build. Against a plain self-hosted
  community instance it fails with a clear error rather than silently
  succeeding; see the `pangolin_user` resource documentation for details.

See individual resource documentation in [docs/](./docs/) for other
mode-specific caveats (e.g. raw tcp/udp resources requiring the
`allow_raw_resources` server flag, or SSH sudo command/group lists requiring
a license/subscription with role-based SSH controls).

## License

This project is licensed under the Mozilla Public License Version 2.0. See the [LICENSE](./LICENSE) file for details.
