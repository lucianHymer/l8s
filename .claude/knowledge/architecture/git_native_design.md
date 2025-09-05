# Git-Native Architecture

L8s has been redesigned as a git extension rather than a standalone container manager, creating a unified Git+SSH+Podman system.

## Core Philosophy

L8s should feel like a natural extension of git workflow, where containers are intrinsically tied to git repositories and worktrees.

## Design Principles

### 1. Git Context Requirement
Most commands REQUIRE git repository context:
- **Git-required commands**: create, ssh, rebuild, rm, exec, push, pull, status
- **Global commands**: init, list, build (work from anywhere)
- No fallback for non-git directories - enforces intended workflow

### 2. Deterministic Container Naming
Container names derived from repository and worktree path:
- Format: `dev-<repo_name>-<worktree_path_hash>`
- Same path always maps to same container
- Each git worktree gets its own container automatically
- No branch coupling - can switch branches within the same container
- Example: `/Users/you/projects/myapp` → `dev-myapp-a3f2d1`

### 3. Worktree→Container Mapping
- Mappings stored in `~/.config/l8s/worktrees.yaml`
- Enables deterministic container resolution from any worktree
- Supports multiple worktrees of the same repository

### 4. Branch Management
- Container automatically checks out the correct branch after push
- Branch that was pushed during creation becomes active
- Fixes issue where containers stayed on default branch

## Current State of Commands

### Commands Requiring Git Context
- **create**: Already checks with IsGitRepository
- **rebuild, rm, ssh, start, stop**: Being updated to require git context
- **push, pull, status**: New commands to be added

### Global Commands
- **init**: System setup, works anywhere
- **list**: Shows all containers with git remote status
- **build**: Binary compilation, works anywhere

## Future Potential

Could eventually become an actual git subcommand (git l8s) for even tighter integration.

## Related Files
- `pkg/cli/handlers.go` - Command implementations
- `pkg/container/manager.go` - Container management logic
- `pkg/git/remote.go` - Git integration
- `GIT_EXTENSION_REQUIREMENTS.md` - Full requirements document