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

## Roadmap (v0.1.0 Deferrals)

The following features are planned for v1.x:

- Raw TCP/UDP resource modes (currently HTTP, SSH, RDP, VNC only)
- Inference mode (AI gateway resources)
- Target health checks
- Fine-grained SSH sudo command/group lists
- Internal (password-based) user support
- User role removal capability

See individual resource documentation in [docs/](./docs/) for details on each deferral.

## License

This project is licensed under the Mozilla Public License Version 2.0. See the [LICENSE](./LICENSE) file for details.
