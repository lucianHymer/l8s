# l8s ZSH Plugin
# Provides command completion for the l8s container management tool

# Add completion function to fpath
fpath=($ZSH_CUSTOM/plugins/l8s $fpath)

# Load the completion function
autoload -U _l8s