# Contributing to terraform-provider-cloudzero

Thank you for your interest in contributing!

## Inbound Licensing

By submitting a contribution to this project, you agree that your contribution
is licensed under the Apache License, Version 2.0, the same license that covers
this project. You retain copyright to your contributions.

## How to Contribute

1. Fork the repository and create a branch from `main`.
2. Make your changes, including tests for any new functionality.
3. Ensure all tests pass: `make test`
4. Ensure lint passes: `make lint`
5. Open a pull request against `main` with a clear description of the change.

## Development Setup

**Requirements:** Go >= 1.24, Terraform >= 1.3

```bash
# Build
make build

# Run unit tests
make test

# Run acceptance tests (requires a CloudZero API key)
export CLOUDZERO_API_KEY="your-api-key"
make testacc
```

## Reporting Security Issues

Please do **not** open a public GitHub issue for security vulnerabilities.
See [SECURITY.md](./SECURITY.md) for responsible disclosure instructions.

## Code of Conduct

Be respectful and constructive. We follow the
[Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
