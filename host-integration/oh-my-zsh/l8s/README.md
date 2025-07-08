# L8s ZSH Plugin

This plugin provides command-line completion for the l8s container management tool.

## Features

- **Command completion**: All l8s commands (init, build, create, list, start, stop, etc.)
- **Dynamic container name completion**: Automatically completes container names from `l8s list`
- **Context-aware filtering**: 
  - `l8s stop` only shows running containers
  - `l8s start` only shows stopped containers
  - `l8s exec` only shows running containers
- **Subcommand completion**: `l8s remote add/remove`
- **Smart suggestions**: Common commands after `l8s exec <container>`

## Installation

### Oh My Zsh

1. The plugin is already in the correct location: `~/.oh-my-zsh/custom/plugins/l8s/`
2. Add `l8s` to your plugins list in `~/.zshrc`:
   ```zsh
   plugins=(... l8s)
   ```
3. Reload your shell: `source ~/.zshrc`

### Manual Installation

1. Add the plugin directory to your `fpath`:
   ```zsh
   fpath=(~/.oh-my-zsh/custom/plugins/l8s $fpath)
   ```
2. Source the plugin:
   ```zsh
   source ~/.oh-my-zsh/custom/plugins/l8s/l8s.plugin.zsh
   ```

## Testing

Run the test suite:
```bash
cd ~/.oh-my-zsh/custom/plugins/l8s/tests
zsh run_all_tests.sh
```

## How It Works

The plugin uses ZSH's powerful completion system to:
1. Parse the current command line
2. Determine what type of completion is needed
3. Query `l8s list` for container names when needed
4. Filter results based on context (running/stopped containers)
5. Provide helpful suggestions

The completion function handles:
- Command names after `l8s`
- Container names after commands that need them
- Subcommands for `l8s remote`
- Common executables after `l8s exec <container>`