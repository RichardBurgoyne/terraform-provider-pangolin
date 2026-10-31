# Terraform Provider for Pangolin

This is a Terraform provider for Pangolin, enabling infrastructure-as-code management of Pangolin resources through Terraform.

## Quick start

```hcl
terraform {
  required_providers {
    pangolin = {
      source = "RichardBurgoyne/pangolin"
    }
  }
}

provider "pangolin" {
  endpoint = "https://api.pangolin.example.com/v1" # or PANGOLIN_ENDPOINT
  api_key  = var.pangolin_api_key                  # or PANGOLIN_API_KEY
}
```

You need a Pangolin instance with the Integration API enabled and an API key created in the dashboard. The `endpoint` includes the version prefix (`/v1`).

The [provider documentation](./docs/index.md) walks through publishing a first service end to end.

## What it manages

| Area | Resources |
|---|---|
| Organization | `pangolin_organization`, `pangolin_domain` |
| Connectivity | `pangolin_site` |
| Public resources | `pangolin_resource`, `pangolin_target`, `pangolin_resource_rule`, `pangolin_resource_mtls`, `pangolin_redirect` |
| Private resources | `pangolin_private_resource` |
| Access | `pangolin_role`, `pangolin_user`, `pangolin_role_resource_grant`, `pangolin_user_resource_grant`, `pangolin_client_role` |
| Organizing | `pangolin_label`, `pangolin_label_attachment` |
| Automation | `pangolin_api_key` |
| AI gateway | `pangolin_ai_provider` |

Data sources: `pangolin_organization`, `pangolin_domain`, `pangolin_site`, `pangolin_resource`, `pangolin_role`, `pangolin_user`.

Tested against Pangolin 1.25. Per-resource documentation, examples and import formats are in [docs/](./docs/).

## Contributing

Contributions are welcome! To build and test the provider locally, run:

```bash
make build    # Build the provider
make test     # Run tests
make lint     # Run linter
make fmt      # Format code
make generate # Regenerate docs/ from the schema, templates/ and examples/
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

- **`pangolin_client_role`** uses routes that only exist in Pangolin Enterprise
  Edition. Clients themselves aren't managed by this provider, so `client_id`
  comes from the dashboard or API.

See individual resource documentation in [docs/](./docs/) for other
mode-specific caveats (e.g. raw tcp/udp resources requiring the
`allow_raw_resources` server flag, or SSH sudo command/group lists requiring
a license/subscription with role-based SSH controls).

## License

This project is licensed under the Mozilla Public License Version 2.0. See the [LICENSE](./LICENSE) file for details.
