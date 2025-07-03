# l8s container zsh configuration
# Path to oh-my-zsh installation
export ZSH="$HOME/.oh-my-zsh"

# Theme
ZSH_THEME="robbyrussell"

# Plugins
plugins=(git golang docker kubectl)

# Load oh-my-zsh
source $ZSH/oh-my-zsh.sh

# Custom aliases
alias ll='ls -la'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias ...='cd ../..'

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
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

# Colored output
export CLICOLOR=1
alias grep='grep --color=auto'
alias fgrep='fgrep --color=auto'
alias egrep='egrep --color=auto'

# FZF configuration if available
if [ -f /usr/share/fzf/shell/key-bindings.zsh ]; then
    source /usr/share/fzf/shell/key-bindings.zsh
fi