# L8s Default Dotfiles

These dotfiles are embedded into the l8s binary and are automatically copied to new containers when no user-specific dotfiles are configured.

## Files Included

- `.zshrc` - Zsh configuration with Oh My Zsh
- `.bashrc` - Bash configuration (fallback shell)
- `.gitconfig` - Git configuration with sensible defaults
- `.tmux.conf` - Tmux configuration

## Customization

Users can override these defaults by:

1. Using `l8s init-dotfiles` to copy these files to `~/.config/l8s/dotfiles/` and customize them
2. Setting the `L8S_DOTFILES` environment variable to point to their dotfiles
3. Using the `--dotfiles-path` flag when creating containers
4. Setting `dotfiles_path` in their l8s config file

## Contributing

When modifying these dotfiles, keep in mind they should work for a wide variety of users and projects. Keep them minimal and focused on developer productivity.