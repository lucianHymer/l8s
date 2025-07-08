# L8s Dotfiles Distribution Plan

## Overview

This document outlines the implementation plan for distributing and managing dotfiles in l8s containers. The goal is to provide a flexible system that works out-of-the-box while allowing user customization and team sharing.

## Architecture

### 1. Dotfile Priority Order (highest to lowest)

1. **Command-line flag**: `--dotfiles-path=/path/to/dotfiles`
2. **Environment variable**: `L8S_DOTFILES=/path/to/dotfiles`
3. **Config file setting**: `dotfiles_path: /path/to/dotfiles`
4. **User dotfiles**: `~/.config/l8s/dotfiles/`
5. **Embedded defaults**: Built into the binary

### 2. File Structure

```
# Embedded in binary (via go:embed)
embed/dotfiles/
├── .zshrc          # Minimal zsh config with oh-my-zsh
├── .bashrc         # Fallback shell config
├── .gitconfig      # Safe git defaults
└── .tmux.conf      # Basic tmux config

# User configuration directory
~/.config/l8s/
├── config.yaml     # Main l8s config
└── dotfiles/       # User's custom dotfiles
    ├── .zshrc
    ├── .vimrc
    └── ...

# Example dotfiles (installed by package manager)
/usr/share/l8s/examples/dotfiles/
├── .zshrc
├── .vimrc
├── .tmux.conf
└── README.md
```

## Implementation Tasks

### Phase 1: Embedded Defaults

1. **Create embed package** (`pkg/embed/`)
   ```go
   //go:embed dotfiles/*
   var DotfilesFS embed.FS
   ```

2. **Add minimal default dotfiles**
   - `.zshrc` with oh-my-zsh and common aliases
   - `.bashrc` as fallback
   - `.gitconfig` with safe defaults
   - `.tmux.conf` with basic settings

3. **Update CopyDotfiles logic**
   - Check each priority level in order
   - Fall back to embedded files if no user dotfiles exist
   - Merge strategy: full file replacement (no partial merging)

### Phase 2: User Customization

1. **Add config file option**
   ```yaml
   # ~/.config/l8s/config.yaml
   dotfiles_path: ~/my-dotfiles
   ```

2. **Add CLI flag**
   ```bash
   l8s create mycontainer --dotfiles-path=~/team-dotfiles
   ```

3. **Create init-dotfiles command**
   ```bash
   l8s init-dotfiles [--template=minimal|full]
   ```
   - Copies example dotfiles to `~/.config/l8s/dotfiles/`
   - User can then customize

### Phase 3: Package Distribution

1. **Homebrew Formula**
   ```ruby
   class L8s < Formula
     # ... build instructions ...
     
     def install
       bin.install "l8s"
       (share/"l8s/examples").install "dotfiles"
     end
   end
   ```

2. **Debian/RPM packages**
   - Binary to `/usr/bin/l8s`
   - Examples to `/usr/share/l8s/examples/`
   - Man pages to `/usr/share/man/`

3. **Container image**
   ```dockerfile
   FROM scratch
   COPY l8s /usr/local/bin/
   COPY examples /usr/share/l8s/examples/
   ```

## Code Changes Required

### 1. Update `pkg/container/manager.go`

```go
func (m *Manager) copyDotfiles(ctx context.Context, containerName string) error {
    // 1. Check command-line flag (if added to command)
    // 2. Check L8S_DOTFILES env var
    // 3. Check config.dotfiles_path
    // 4. Check ~/.config/l8s/dotfiles/
    // 5. Use embedded defaults
    
    dotfilesDir := m.getDotfilesPath()
    if dotfilesDir == "" {
        // Use embedded defaults
        return m.copyEmbeddedDotfiles(ctx, containerName)
    }
    
    // Copy user dotfiles
    return CopyDotfilesToContainer(ctx, m.client, containerName, dotfilesDir, m.config.ContainerUser)
}
```

### 2. Create `pkg/embed/dotfiles.go`

```go
package embed

import (
    "embed"
    "io/fs"
)

//go:embed dotfiles/.* dotfiles/*
var DotfilesFS embed.FS

func GetDotfilesFS() (fs.FS, error) {
    return fs.Sub(DotfilesFS, "dotfiles")
}
```

### 3. Add to `pkg/config/config.go`

```go
type Config struct {
    // ... existing fields ...
    
    // DotfilesPath specifies custom dotfiles location
    DotfilesPath string `yaml:"dotfiles_path,omitempty"`
}
```

### 4. Create `cmd/commands/init_dotfiles.go`

```go
func InitDotfilesCmd() *cobra.Command {
    var template string
    
    cmd := &cobra.Command{
        Use:   "init-dotfiles",
        Short: "Initialize user dotfiles from templates",
        RunE: func(cmd *cobra.Command, args []string) error {
            return initializeDotfiles(template)
        },
    }
    
    cmd.Flags().StringVar(&template, "template", "minimal", "Template to use (minimal|full)")
    return cmd
}
```

## Migration Guide

### For Current Users

1. If using repository dotfiles:
   ```bash
   # Option 1: Copy to user config
   cp -r ./dotfiles ~/.config/l8s/
   
   # Option 2: Set environment variable
   export L8S_DOTFILES=/path/to/repo/dotfiles
   
   # Option 3: Update config file
   echo "dotfiles_path: /path/to/repo/dotfiles" >> ~/.config/l8s/config.yaml
   ```

### For Package Maintainers

1. Ensure examples are installed to standard location
2. Don't install user-specific files to `~/.config/`
3. Include man pages and shell completions

## Testing Strategy

1. **Unit tests** for dotfile resolution logic
2. **Integration tests** for each priority level
3. **Manual testing** of package installations
4. **Upgrade testing** to ensure user configs aren't overwritten

## Documentation Updates

1. **README.md**: Add dotfiles customization section
2. **Man page**: Document all dotfiles options
3. **Examples**: Provide sample dotfiles for different workflows
4. **FAQ**: Common dotfiles questions

## Success Criteria

- [ ] Zero-config: Works immediately after `brew install l8s`
- [ ] Customizable: Users can easily override defaults
- [ ] Team-friendly: Can share dotfiles via git
- [ ] Package-ready: Works with standard package managers
- [ ] Backward compatible: Existing users' workflows continue working

## Timeline

- **Week 1**: Implement embedded defaults (Phase 1)
- **Week 2**: Add user customization options (Phase 2)
- **Week 3**: Test packaging and distribution (Phase 3)
- **Week 4**: Documentation and migration support