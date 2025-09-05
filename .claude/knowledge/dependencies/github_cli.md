# GitHub CLI in L8s Containers

GitHub CLI (gh) is installed in all L8s containers to enable seamless development workflows.

## Installation Process

The GitHub CLI is installed in the Containerfile during Section 4 (Full Package Installation):

1. **dnf5-plugins package** - Required for dnf config-manager
2. **GitHub's official RPM repository** - Added to dnf repos
3. **gh package** - Installed from the gh-cli repo

This ensures containers have the latest GitHub CLI version for creating PRs, managing issues, and other GitHub operations.

## Integration with L8s

The GitHub CLI works seamlessly with:
- GitHub tokens configured via `l8s init` or config file
- Automatic GITHUB_TOKEN environment variable injection
- Repository operations within development containers

## Related Files
- `pkg/embed/containers/Containerfile` - Container image definition with gh installation