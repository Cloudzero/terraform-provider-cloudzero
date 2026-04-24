# Terraform Provider — Security Setup Request

## What is this?

We have built a Terraform provider that allows customers to manage CloudZero resources (Views, Budgets, Insights) and connect AWS accounts as code. To publish it on the [Terraform Registry](https://registry.terraform.io), we need the following from the security/platform team.

## Why does this matter?

The lack of a Terraform provider was raised in 10+ prospect calls with $230K ARR directly lost and $990K+ ARR exposure in open deals. Engineering-led buyers expect everything-as-code — without a provider, CloudZero is positioned as a "FinOps team tool" rather than a "developer platform."

## Current state

- Provider code: [Cloudzero/project-terraform-provider-cloudzero](https://github.com/Cloudzero/project-terraform-provider-cloudzero) (private)
- CI: all checks passing (build, lint, test, generate)
- End-to-end tested: single `terraform apply` creates IAM role + registers AWS account with CloudZero
- No new API endpoints were needed — uses existing `/accounts/v1/link`

## What we need

### 1. Rename the GitHub repo

**Current:** `Cloudzero/project-terraform-provider-cloudzero`
**Required:** `Cloudzero/terraform-provider-cloudzero`

The Terraform Registry requires the repo name to follow the pattern `terraform-provider-{NAME}`. The CloudZero GitHub org naming policy (`^(feature|library|cz|...)`) blocks this — an admin needs to rename it or create an exception.

**GitHub:** Settings > General > Repository name

### 2. Make the repo public

The Terraform Registry only indexes public repositories. The repo must be public before publishing.

**GitHub:** Settings > General > Danger Zone > Change visibility

### 3. Generate a GPG signing key

Every Terraform provider release must be signed. The Registry verifies the signature against a registered public key.

**Requirements:**
- Algorithm: **RSA 4096** (ECC is not supported by the Registry)
- Suggested identity: `CloudZero Engineering <engineering@cloudzero.com>`
- The private key and passphrase must be stored securely
- Consider a key rotation policy

**Commands:**
```bash
# Generate the key (interactive — prompts for passphrase)
gpg --full-generate-key
# Select: (1) RSA and RSA, 4096 bits, does not expire

# Export the private key (for GitHub secret)
gpg --armor --export-secret-keys <KEY_ID>

# Export the public key (for Terraform Registry)
gpg --armor --export <KEY_ID>
```

### 4. Add GitHub repository secrets

Two secrets are needed on the repo for the release workflow:

| Secret name | Value | How to get it |
|---|---|---|
| `GPG_PRIVATE_KEY` | ASCII-armored private key | `gpg --armor --export-secret-keys <KEY_ID>` |
| `PASSPHRASE` | GPG key passphrase | The passphrase entered during key generation |

**GitHub:** Repo > Settings > Secrets and variables > Actions > New repository secret

### 5. Register the GPG public key on the Terraform Registry

1. Go to [registry.terraform.io](https://registry.terraform.io)
2. Sign in with a GitHub account that is an admin of the CloudZero org
3. Navigate to **Settings > Signing Keys**
4. Add the ASCII-armored public key (`gpg --armor --export <KEY_ID>`)

### 6. Publish the provider

After steps 1-5 are complete:

1. Tag the release: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions will automatically build binaries, sign checksums, and create a GitHub Release
3. Go to [registry.terraform.io/publish/provider](https://registry.terraform.io/publish/provider)
4. Select the `terraform-provider-cloudzero` repository
5. The Registry creates a webhook — future releases sync automatically

## References

- [Terraform Registry publishing docs](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [GoReleaser GitHub Actions workflow](https://goreleaser.com/ci/actions/) (already configured in the repo)
- Provider repo: https://github.com/Cloudzero/project-terraform-provider-cloudzero
- Companion module: https://github.com/Cloudzero/provision-account/tree/master/terraform/cloudzero-aws

## Questions?

Reach out to Kevin Lamb or the platform engineering team.
