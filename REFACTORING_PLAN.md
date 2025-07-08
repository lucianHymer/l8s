# L8S Codebase Refactoring Plan

## Overview

The l8s codebase currently has duplicate command implementations due to an incomplete migration from standalone commands to a factory pattern. This document outlines the steps needed to complete the refactoring and clean up the codebase.

## Current State

### Architecture Issues

1. **Duplicate Implementations**
   - Dead code in `cmd/commands/*.go` (except `init.go`)
   - Live code in `pkg/cli/handlers.go`
   - Features were accidentally added to the dead code instead of the live code

2. **Inconsistent Patterns**
   - Most commands use the factory pattern (`pkg/cli/`)
   - The `init` command is called directly from `cmd/commands/`
   - Business logic exists in both `cmd/` and `pkg/` directories

3. **Missing Features in Live Code**
   - Color support (added to dead code by mistake)
   - Command aliases (`ls` for list, `rm` for remove)
   - Remove command flags (`--force`, `--keep-volumes`)
   - SSH key validation in create command
   - Better status messages and formatting

## Refactoring Steps

### Phase 1: Port Missing Features (Priority: High)

1. **Port Color Support to Live Code**
   - [ ] Add color imports to `pkg/cli/handlers.go`
   - [ ] Port `formatStatus()` and `formatGitStatus()` functions from dead code
   - [ ] Update all command outputs to use color formatting
   - [ ] Ensure NO_COLOR environment variable is respected

2. **Add Command Aliases**
   - [ ] Add `Aliases: []string{"ls"}` to ListCmd in factory
   - [ ] Add `Aliases: []string{"rm"}` to RemoveCmd in factory

3. **Enhance Remove Command**
   - [ ] Add `--force` flag to skip confirmation prompt
   - [ ] Add `--keep-volumes` flag to preserve volumes
   - [ ] Update `runRemove` handler to respect these flags

4. **Add SSH Key Validation**
   - [ ] Port `ssh.ValidatePublicKey()` call from dead create command
   - [ ] Add validation before container creation

### Phase 2: Migrate Init Command (Priority: High)

1. **Move Init to Factory Pattern**
   - [ ] Create `runInit` method in `pkg/cli/handlers.go`
   - [ ] Move logic from `cmd/commands/init.go` to the handler
   - [ ] Create `InitCmd()` method in factory
   - [ ] Update `main.go` to use `factory.InitCmd()` instead of direct import

2. **Handle Init's Special Requirements**
   - [ ] Ensure init command can run without existing config
   - [ ] Preserve interactive prompts and remote server setup
   - [ ] Maintain all validation logic

### Phase 3: Clean Up Dead Code (Priority: Medium)

1. **Remove Dead Command Files**
   - [ ] Delete `cmd/commands/list.go`
   - [ ] Delete `cmd/commands/create.go`
   - [ ] Delete `cmd/commands/remove.go`
   - [ ] Delete `cmd/commands/start.go`
   - [ ] Delete `cmd/commands/stop.go`
   - [ ] Delete `cmd/commands/info.go`
   - [ ] Delete `cmd/commands/ssh.go`
   - [ ] Delete `cmd/commands/init_dotfiles.go` (completely unused)
   - [ ] Delete `cmd/commands/remote.go` (if handlers exist)
   - [ ] Delete `cmd/commands/build.go` (if handlers exist)

2. **Clean Up Test Files**
   - [ ] Delete `/test_color.go`
   - [ ] Delete `/test_color2.go`
   - [ ] Delete `/test_list_color.go`
   - [ ] Delete `/debug_color.go`
   - [ ] Delete `/force_color.go`
   - [ ] Delete `/colortest/` directory

3. **Remove Dead Code Import**
   - [ ] Remove `"github.com/l8s/l8s/cmd/commands"` import from `main.go`

### Phase 4: Improve Architecture (Priority: Low)

1. **Standardize Command Structure**
   - [ ] Ensure all commands follow the same pattern
   - [ ] Document the factory pattern usage
   - [ ] Add unit tests for command handlers

2. **Consider Init-Dotfiles Command**
   - [ ] Evaluate if init-dotfiles should be added to live code
   - [ ] If yes, implement in factory pattern
   - [ ] If no, ensure it's completely removed

## Testing Plan

After each phase:
1. Build the project: `go build ./cmd/l8s`
2. Test each command manually
3. Verify color output works correctly
4. Check that aliases work (`l8s ls`, `l8s rm`)
5. Test new flags (`--force`, `--keep-volumes`)
6. Ensure init command still works

## Success Criteria

- [ ] No duplicate command implementations
- [ ] All commands use the factory pattern consistently
- [ ] Color support works in terminal
- [ ] All features from dead code are available in live code
- [ ] No dead code files remain
- [ ] Clean `git status` with no uncommitted debug code

## Notes for Implementers

- The color package already exists at `pkg/color/color.go`
- When porting color support, make sure to use the package's functions that check `isColorEnabled()`
- The factory pattern uses dependency injection for testing - maintain this approach
- Be careful with the init command migration - it has special requirements since it runs before config exists
- Test thoroughly - the dead code had features that users might be expecting