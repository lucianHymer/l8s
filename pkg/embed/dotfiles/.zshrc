# l8s container zsh configuration
# Path to oh-my-zsh installation
export ZSH="$HOME/.oh-my-zsh"

# Disable oh-my-zsh theme since we're using oh-my-posh
ZSH_THEME=""

# Plugins
plugins=(git golang)

# Load oh-my-zsh
source $ZSH/oh-my-zsh.sh

# Initialize oh-my-posh
if command -v oh-my-posh &> /dev/null; then
    eval "$(oh-my-posh init zsh --config ~/.config/ohmyposh/config.json)"
fi

# Custom aliases
alias ll='ls -lAtr'
alias v='nvim'
alias danger='claude --dangerously-skip-permissions'


# Git aliases
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline --graph --decorate'
alias gd='git diff'

# Development aliases
alias m='make'
alias mt='make test'
alias mb='make build'

# Set editor
export EDITOR=nvim

# Better history
HISTSIZE=10000
SAVEHIST=10000
setopt SHARE_HISTORY
setopt HIST_IGNORE_DUPS
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_FIND_NO_DUPS

# Go development
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct

# Path additions
export PATH=$HOME/.local/bin:$PATH:/usr/local/go/bin:$HOME/go/bin

# Colored output
export CLICOLOR=1
alias grep='grep --color=auto'
alias fgrep='fgrep --color=auto'
alias egrep='egrep --color=auto'

# FZF configuration if available
if [ -f /usr/share/fzf/shell/key-bindings.zsh ]; then
    source /usr/share/fzf/shell/key-bindings.zsh
fi

# Team command for dtach session management
# The team command is now a standalone script in ~/.local/bin/team
# Use 'team <name>' to create/attach to persistent sessions
# Use 'team list' to see active sessions
# Sessions persist across SSH disconnections

# ZSH completion for team command
_team() {
    local -a subcmds sessions
    subcmds=('ls:List active sessions' 'attach:Attach to existing session' 'create:Create new session' 'help:Show help')

    if (( CURRENT == 2 )); then
        _describe 'command' subcmds
    elif (( CURRENT == 3 )) && [[ "$words[2]" == "attach" ]]; then
        # Complete session names
        sessions=()
        for sock in /tmp/dtach-*.sock(N); do
            if [[ -S "$sock" ]]; then
                local basename=${sock:t}
                local encoded=${basename#dtach-}
                encoded=${encoded%.sock}
                local name=$(echo "$encoded" | base64 -d 2>/dev/null)
                [[ -n "$name" ]] && sessions+=("$name")
            fi
        done
        _values 'session' $sessions
    fi
}
compdef _team team

# Ripgrep + FZF + Neovim integration
# Usage: r <pattern> [rg options]
# Search for pattern with ripgrep, preview with bat, open in neovim
r() {
  local result file line
  result=$(
    rg --line-number --no-heading --color=always "$@" | \
      fzf --ansi \
          --delimiter ':' \
          --preview 'bat --color=always --highlight-line {2} {1}' \
          --preview-window '+{2}/2'
  )

  [[ -z "$result" ]] && return

  file=$(echo "$result" | cut -d: -f1)
  line=$(echo "$result" | cut -d: -f2)

  nvim +"$line" "$file"
}

cd /workspace/project
