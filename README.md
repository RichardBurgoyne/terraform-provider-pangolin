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

## License

This project is licensed under the Mozilla Public License Version 2.0. See the [LICENSE](./LICENSE) file for details.
