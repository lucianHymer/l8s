# Git Remote Flow for L8s Create Command

## The Problem

L8s creates isolated development containers that developers can SSH into. Currently, the `create` command requires a git URL to clone a repository into the container:

```bash
l8s create mycontainer https://github.com/user/repo.git
```

This has several issues:

1. **Authentication headaches** - Private repos require tokens/SSH keys to be configured
2. **Protocol mismatch** - Local repos often use SSH, but containers need HTTPS clones
3. **Redundant** - Usually creating a container for the repo you're already working in

## The Solution

**Remove the git URL requirement and instead:**

1. Only allow `create` from within a git repository
2. Set up an empty git repo in the container that can receive pushes
3. Automatically push the current branch (or --branch) HEAD to populate the container
4. Automatically add the container as a git remote to the local repo
5. Developers can continue pulling changes from the container as they work (or occasionally pushing to the container)

**New workflow:**
```bash
# From within your project directory
l8s create mycontainer              # Creates and pushes current branch HEAD
l8s create mycontainer --branch=feature  # Creates and pushes feature branch HEAD

# Later, pull changes
git pull mycontainer main
```

## Why This Works

### Security & Isolation
- Container never needs access to the host machine
- Only committed code enters the container (no .env files or secrets accidentally copied)
- All communication is one-way: host → container via SSH

### Developer Experience
- No authentication setup needed
- Works with any git hosting (GitHub, GitLab, self-hosted)
- Natural git workflow developers already understand
- Can push different branches without recreating containers

### Technical Implementation

The key insight is using git's `receive.denyCurrentBranch updateInstead` configuration, which:
- Allows pushing to a non-bare repository
- Automatically updates the working tree when receiving pushes
- Prevents pushes that would overwrite uncommitted changes

## Implementation Guide

### Phase 1: Modify CLI Command

**Current command:**
```
l8s create <name> <git-url> [branch]
```

**New command:**
```
l8s create <name> [--branch=<branch>]
```

The command now:
1. Checks if you're in a git repository
2. Gets the current branch (or uses --branch flag)
3. Creates container with empty git repo
4. Adds container as a git remote
5. Pushes the HEAD of the specified branch to populate the container

### Phase 2: Container Setup

Instead of cloning a repository, the container initialization:

```bash
# Create empty git repository
git init /workspace/project

# Configure it to accept pushes and update working tree
git -C /workspace/project config receive.denyCurrentBranch updateInstead

# Set ownership (should already be handled elsewhere, and should use correct user)
chown -R dev:dev /workspace/project
```

### Phase 3: Git Remote Management and Initial Push

The container manager needs to:

```go
// When creating container
func (m *Manager) addGitRemote(name, containerName string, sshPort int) error {
    // Build SSH URL: ssh://dev@localhost:2222/workspace/project
    remoteURL := fmt.Sprintf(...)
    
    // Add remote to current git repo
    return git.AddRemote(".", name, remoteURL)
}

// Push initial code to container
func (m *Manager) pushInitialCode(name, branch string) error {
    // Push the HEAD of the specified branch
    cmd := exec.Command("git", "push", name, fmt.Sprintf("%s:%s", branch, branch))
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to push initial code: %w\nOutput: %s", err, string(output))
    }
    return nil
}

// When removing container
func (m *Manager) removeGitRemote(name string) error {
    // Remove remote if we're still in a git repo
    return git.RemoveRemote(".", name)
}
```

## User Experience

### Happy Path
```bash
$ cd my-project
$ l8s create dev
🎳 Creating container: l8s-dev
✓ SSH port: 2222
✓ Git remote 'dev' added
✓ Pushed main branch (HEAD: abc123) to container
✓ Container ready with your code

$ # Make some changes and push again
$ git add . && git commit -m "Add feature"
$ git push dev main
To ssh://dev@localhost:2222/workspace/project
   abc123..def456  main -> main
```

### Safety Features

If you try to push while there are uncommitted changes in the container:

```bash
$ git push dev main
To ssh://dev@localhost:2222/workspace/project
 ! [remote rejected] main -> main (Working tree has unstaged changes)
error: failed to push some refs to 'ssh://dev@localhost:2222/workspace/project'
```

This prevents accidentally losing work in the container.

## Migration Strategy

All new, don't worry about it.

## Benefits Summary

1. **Simpler** - No URL validation, no auth configuration
2. **Safer** - Can't overwrite uncommitted changes
3. **More flexible** - Push any branch anytime
4. **Better isolation** - Only committed code enters containers
5. **Familiar** - Uses standard git commands developers know

## Considerations

- Must run `l8s create` from within a git repository
- Container is immediately populated with the HEAD of the specified branch
- Only committed code at HEAD is pushed (uncommitted changes stay local)
- Subsequent pushes follow normal git push rules (can be rejected if container has changes)
