# Containerfiles Location

**Date**: 2025-10-29

## Issue

The Containerfiles are not in a top-level `pkg/containerfiles/` directory as one might expect.

## Actual Location

Containerfiles are embedded with other resources in:
- **Main Containerfile**: `pkg/embed/containers/Containerfile`
- **Directory**: `pkg/embed/containers/`

## Why This Location?

The Containerfiles are treated as embedded resources that get built into the binary, similar to dotfiles. This allows L8s to build container images without requiring external file dependencies.

## Related Files
- `pkg/embed/containers/Containerfile` - Main container image definition
