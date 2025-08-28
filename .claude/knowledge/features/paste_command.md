# Paste Command Feature

The paste command transfers clipboard content from macOS to remote containers, enabling easy data sharing.

## Command Usage

```bash
l8s paste <container> [name]
```

Transfers clipboard content to `/tmp/claude-clipboard/` in the container.

## Implementation Details

### Platform Support
- **macOS-only** initially
- Uses `osascript` for image detection and extraction
- Uses `pbpaste` for text content

### File Naming Strategy
- **Default files**: `clipboard.png` or `clipboard.txt` (always replaced)
- **Named files**: `clipboard-<name>.png` or `clipboard-<name>.txt` (preserved)
- Automatic format detection based on clipboard content

### Architecture

Follows L8s dual factory pattern:
- LazyCommandFactory for command registration
- CommandFactory for implementation
- Uses `ExecContainerWithInput` for content transfer via `tee` command

### Claude Integration

Includes `/paste` slash command in `.claude/commands/paste.md`:
- Detects recently pasted files
- Prompts for analysis of pasted content
- Integrates with Claude Code workflow

## Technical Components

- `pkg/cli/clipboard.go` - Clipboard detection and extraction utilities
- `pkg/cli/handlers.go` - runPaste handler implementation
- `pkg/cli/factory_lazy.go` - PasteCmd command definition
- `pkg/embed/dotfiles/.claude/commands/paste.md` - Claude slash command
- Added `ExecContainerWithInput` to ContainerManager interface

## Transfer Method

1. Detect clipboard type (image vs text)
2. Extract content using platform-specific tools
3. Create target directory in container
4. Pipe content through `tee` command for writing
5. Report success with file path

## Future Enhancements

- Linux clipboard support (xclip/xsel)
- Windows clipboard support
- Multiple file formats
- Clipboard history management