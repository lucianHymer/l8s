# Slash Commands

L8s containers include custom slash commands that integrate with Claude Code for enhanced development workflows.

## /req Command

A slash command for reading and providing context from requirements files to Claude.

### Implementation Details
- **Location**: `pkg/embed/dotfiles/.claude/commands/req.md`
- **Purpose**: Reads specified requirements files to provide Claude with project context
- **Usage**: `/req <file_path> [optional clarification text]`

### Command Structure
1. Accepts file path as first argument ($1)
2. Optional additional clarification text as remaining arguments ($ARGUMENTS)
3. Reads the specified requirements file
4. Provides context to Claude for understanding project requirements

### Technical Notes
- Follows the pattern of existing commands like paste.md
- Uses frontmatter for allowed-tools configuration
- Uses bash commands with ! prefix for dynamic file reading (not @ syntax)

## /paste Command

Existing slash command for clipboard operations (see paste_command.md for details).

## Related Files
- `pkg/embed/dotfiles/.claude/commands/req.md` - Requirements reading command
- `pkg/embed/dotfiles/.claude/commands/paste.md` - Clipboard paste command