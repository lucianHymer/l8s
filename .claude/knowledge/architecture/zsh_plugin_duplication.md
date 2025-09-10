# ZSH Plugin Duplicate Locations

L8s maintains two identical copies of the ZSH plugin that must be kept in sync.

## Plugin Locations

### 1. Original Source
**Path**: `host-integration/oh-my-zsh/l8s/`

This is the primary development location for the ZSH plugin, containing:
- `_l8s` - Main completion file
- Test suite and framework
- Plugin documentation

### 2. Embedded Copy
**Path**: `pkg/embed/host-integration/oh-my-zsh/l8s/`

This is the embedded version that:
- Gets bundled into the l8s binary
- Is extracted via `l8s install-zsh-plugin` command
- Installed to `~/.oh-my-zsh/custom/plugins/l8s`

## Synchronization Requirement

**Critical**: Both copies must be kept identical when making changes to the ZSH plugin. Any modifications to the completion logic, flags, or behavior must be applied to both locations.

## Installation Flow

1. Developer modifies plugin in `host-integration/oh-my-zsh/l8s/`
2. Same changes applied to `pkg/embed/host-integration/oh-my-zsh/l8s/`
3. Binary rebuilt with updated embedded plugin
4. Users run `l8s install-zsh-plugin` to get latest version

## Related Files
- `pkg/embed/host_integration.go` - Handles plugin extraction
- `host-integration/oh-my-zsh/l8s/_l8s` - Original plugin
- `pkg/embed/host-integration/oh-my-zsh/l8s/_l8s` - Embedded copy