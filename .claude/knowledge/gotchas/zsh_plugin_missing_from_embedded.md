# ZSH Plugin Missing from Embedded Dotfiles

**Date**: 2025-09-10

## Issue

When the dotfiles were reorganized to separate host integration from container dotfiles, the .oh-my-zsh/custom/plugins/l8s directory was moved from dotfiles/ to host-integration/, but the container still needs the ZSH plugin files to be embedded for tab completion to work inside containers.

## Root Cause

The files were moved in commit 9031fcf but the container-side plugin files may have been lost in the process. The ZSH plugin was relocated from the container dotfiles to host integration, breaking container tab completion.

## Solution

The ZSH plugin needs to be embedded in containers for tab completion to work properly inside the development environments.

## Related Files
- `pkg/embed/dotfiles/` - Container dotfiles location
- `host-integration/oh-my-zsh/l8s/` - Host ZSH plugin location