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
team() {
    local cmd="$1"
    
    # Show help for various help flags or no arguments at all
    if [ -z "$cmd" ] || [[ "$cmd" =~ ^(-h|--help|-help|help)$ ]]; then
        cat <<EOF
team - Manage dtach sessions for team collaboration

Usage:
  team <name>              Create or attach to a session
  team ls|list             List active sessions
  team attach <name>       Attach to existing session (read-only)
  team create <name>       Create new session (same as 'team <name>')
  team help                Show this help message

Examples:
  team frontend            Create/attach to 'frontend' session
  team ls                  Show all active sessions
  team attach backend      Attach to existing 'backend' session

Sessions persist across connections. Use Ctrl+\ to detach.
EOF
        return 0
    fi
    
    case "$cmd" in
        ls|list)
            echo "Active sessions:"
            local found=0
            for sock in /tmp/dtach-*.sock; do
                if [ -S "$sock" ]; then
                    # Extract encoded name from socket path
                    local encoded=$(basename "$sock" .sock | sed 's/^dtach-//')
                    local name=$(echo "$encoded" | base64 -d 2>/dev/null || echo "invalid")
                    echo "  $name"
                    found=1
                fi
            done
            [ $found -eq 0 ] && echo "  (none)"
            ;;
        
        attach|a)
            local name="$2"
            if [ -z "$name" ]; then
                echo "Usage: team attach <name>"
                return 1
            fi
            local encoded=$(echo -n "$name" | base64 | tr -d '\n')
            local socket="/tmp/dtach-$encoded.sock"
            if [ -S "$socket" ]; then
                DTACH_SESSION="$name" dtach -a "$socket"
            else
                echo "Session '$name' not found"
                echo "Active sessions:"
                team ls | tail -n +2  # Skip the "Active sessions:" header
                return 1
            fi
            ;;
        
        create|c)
            local name="$2"
            if [ -z "$name" ]; then
                echo "Usage: team create <name>"
                return 1
            fi
            local encoded=$(echo -n "$name" | base64 | tr -d '\n')
            local socket="/tmp/dtach-$encoded.sock"
            DTACH_SESSION="$name" dtach -A "$socket" -z /bin/zsh
            ;;
        
        *)
            # Default to create if just given a name
            local name="$cmd"
            local encoded=$(echo -n "$name" | base64 | tr -d '\n')
            local socket="/tmp/dtach-$encoded.sock"
            DTACH_SESSION="$name" dtach -A "$socket" -z /bin/zsh
            ;;
    esac
}

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
                local encoded=${${sock:t}%.sock#dtach-}
                local name=$(echo "$encoded" | base64 -d 2>/dev/null)
                [[ -n "$name" ]] && sessions+=("$name")
            fi
        done
        _values 'session' $sessions
    fi
}
compdef _team team
