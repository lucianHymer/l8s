# L8s Host Integration

This directory contains files and plugins for your HOST machine (where you run l8s commands), not for the containers.

## Oh My Zsh Plugin

The `oh-my-zsh/l8s` directory contains a ZSH plugin that provides command completion for l8s.

### Installation

1. Copy the plugin to your oh-my-zsh custom plugins directory:
   ```bash
   cp -r oh-my-zsh/l8s ~/.oh-my-zsh/custom/plugins/
   ```

2. Add `l8s` to your plugins list in `~/.zshrc`:
   ```bash
   plugins=(... l8s)
   ```

3. Reload your shell:
   ```bash
   source ~/.zshrc
   ```

### Features

- Command completion for all l8s subcommands
- Container name completion for commands that take container names
- Branch completion for the create command
- Context-aware filtering (only shows relevant options)

## Shell Completions

Future shell completion scripts for other shells (bash, fish, etc.) will be added here.

## Note

These files are for your development machine, not for the containers. Container dotfiles are embedded in the l8s binary and can be customized using `l8s init-dotfiles`.