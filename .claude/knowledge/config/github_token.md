# GitHub Token Configuration

L8s supports GitHub fine-grained personal access tokens for secure development workflows.

## Token Types

L8s uses **fine-grained personal access tokens** (not classic tokens) for better security:
- Start with `github_pat_` prefix (not `ghp_` like classic tokens)
- Allow per-repository or per-organization scoping
- Support granular permissions per resource type
- Have expiration dates (max 1 year)
- Better audit logging and security

## Recommended Permissions

For optimal L8s development workflows, configure your token with:

| Permission | Access Level | Purpose |
|------------|-------------|---------|
| **Actions** | Read-only | View workflow runs and artifacts |
| **Contents** | Read-only | Access repository contents, commits, branches |
| **Issues** | Read and write | Create/edit issues and comments |
| **Metadata** | Read-only | Required - search repos and access metadata |
| **Pull requests** | Read and write | Create/edit PRs and related comments |

These permissions enable productive GitHub interaction while maintaining security through mostly read-only access.

## Configuration

### Token Creation
Create a token at: https://github.com/settings/personal-access-tokens/new

### Setting the Token
Configure during initialization:
```bash
l8s init  # Prompts for GitHub token
```

Or manually in `~/.config/l8s/config.yaml`:
```yaml
github_token: github_pat_...
```

### Container Integration
- Token automatically injected as `GITHUB_TOKEN` environment variable
- Available in container's `.zshrc`
- Used by GitHub CLI for authentication
- Can be overridden per-container by editing `.zshrc`

## Security Best Practices

1. **Scope tokens narrowly** - Only to repositories/orgs you're working with
2. **Set expiration dates** - Maximum 1 year, shorter is better
3. **Use minimal permissions** - Only what's needed for development
4. **Rotate regularly** - Replace tokens periodically
5. **Never commit tokens** - Keep them out of version control

## Related Files
- `pkg/cli/handlers.go` - Init command token configuration
- `pkg/config/config.go` - Configuration structure with token field