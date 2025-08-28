# L8s Paste Feature Specification

## Overview

Add clipboard sharing functionality to l8s that allows users to paste images or text from their local machine's clipboard into remote development containers. This feature enables seamless sharing of screenshots, code snippets, and other clipboard content with containerized development environments where Claude Code or other tools are running.

## Use Case

**Problem:** Developers need to share screenshots or text from their local machine with tools running inside remote containers (e.g., Claude Code for visual debugging or code analysis).

**Solution:** A simple `l8s paste` command that transfers clipboard contents to a predictable location in the container, paired with Claude Code hooks for streamlined workflows.

## Command Specification

### Command Syntax
```bash
l8s paste <container> [name]
```

### Parameters
- `<container>` (required): Target container name (e.g., `dev-embed`)
- `[name]` (optional): Custom name for the pasted file

### Behavior

#### Default Mode (no name provided)
1. Ensure directory exists: `mkdir -p /tmp/claude-clipboard`
2. Detect clipboard content type (image or text)
3. Delete existing default files: `/tmp/claude-clipboard/clipboard.png` and `/tmp/claude-clipboard/clipboard.txt`
4. Create new file based on content type:
   - Image → `/tmp/claude-clipboard/clipboard.png`
   - Text → `/tmp/claude-clipboard/clipboard.txt`
5. Output the destination path to stdout

#### Named Mode (name provided)
1. Ensure directory exists: `mkdir -p /tmp/claude-clipboard`
2. Detect clipboard content type (image or text)
3. Create file with custom name:
   - Image → `/tmp/claude-clipboard/clipboard-{name}.png`
   - Text → `/tmp/claude-clipboard/clipboard-{name}.txt`
4. Leave default files untouched
5. Output the destination path to stdout

### Output Format
```bash
# Success
Pasted to /tmp/claude-clipboard/clipboard.png

# With name
Pasted to /tmp/claude-clipboard/clipboard-login-bug.png

# Error
Error: No image or text found in clipboard
Error: Container 'dev-embed' not found
Error: Failed to paste: <reason>
```

## Implementation Details

### Platform Support

**Current Support: macOS only**

The initial implementation only supports macOS. The command should check the runtime platform and return an error on non-Mac systems:

```go
func checkPlatform() error {
    if runtime.GOOS != "darwin" {
        return fmt.Errorf("paste command is currently only supported on macOS")
    }
    return nil
}
```

#### macOS Implementation
```bash
# Detect image in clipboard
osascript -e 'clipboard info' 2>/dev/null | grep -q 'PNGf\|JPEG\|TIFF'

# Extract image from clipboard
osascript -e 'set thePNG to the clipboard as «class PNGf»' \
          -e 'set theFile to open for access POSIX file "/tmp/local-clipboard.png" with write permission' \
          -e 'write thePNG to theFile' \
          -e 'close access theFile'

# Extract text from clipboard
pbpaste > /tmp/local-clipboard.txt

# Transfer to container
cat /tmp/local-clipboard.png | ssh <container-ssh> 'mkdir -p /tmp/claude-clipboard && cat > /tmp/claude-clipboard/clipboard.png'
```

#### Future Platform Support
- **Linux**: Would require xclip (X11) or wl-paste (Wayland)
- **Windows**: Would require PowerShell Get-Clipboard cmdlet

### Content Type Detection Logic

1. First, check for image content
2. If no image found, check for text content
3. If neither found, return error

```go
func detectClipboardType() (ContentType, error) {
    // Platform-specific detection
    if hasImageContent() {
        return ImageContent, nil
    }
    if hasTextContent() {
        return TextContent, nil
    }
    return Unknown, fmt.Errorf("no supported content in clipboard")
}
```

### File Management

- All files are stored in `/tmp/claude-clipboard/` directory
- Directory is created if it doesn't exist
- Default files (`/tmp/claude-clipboard/clipboard.png`, `/tmp/claude-clipboard/clipboard.txt`) are mutually exclusive
- Named files can coexist with defaults and each other
- File permissions should be readable by the container user (typically `dev`)

### Error Handling

1. **No clipboard content**: Exit with error message
2. **Container not running**: Exit with helpful error
3. **SSH failure**: Show SSH error and suggest checking connection
4. **Platform not supported**: Show message about platform limitations

## Claude Code Integration

### Custom Slash Command: `/paste`

**Purpose**: Tell Claude to analyze the most recently pasted clipboard content

**Installation**: Create the file `pkg/embed/dotfiles/.claude/commands/paste.md` with the following content:

**Implementation with file detection**:
```markdown
---
allowed-tools: Bash(test:/tmp/claude-clipboard/*), Bash(echo:*)
---
!if [ -f /tmp/claude-clipboard/clipboard.png ]; then echo "Screenshot pasted to /tmp/claude-clipboard/clipboard.png"; elif [ -f /tmp/claude-clipboard/clipboard.txt ]; then echo "Text pasted to /tmp/claude-clipboard/clipboard.txt"; else echo "No pasted content found"; fi

Please analyze the pasted content mentioned above.

$ARGUMENTS
```

**How it works**:
1. User runs `l8s paste <container>` from their Mac to paste clipboard content
2. User types `/paste` in Claude Code with optional prompt
3. The bash command runs first and injects its output
4. Claude receives both the bash output and the instruction
5. Claude reads and analyzes the appropriate file

**Example with arguments**:
```markdown
# User types: /paste What's wrong with this UI?
# Claude sees:
# Screenshot pasted to /tmp/claude-clipboard/clipboard.png
# Please analyze the pasted content mentioned above.
# What's wrong with this UI?
```

### Usage Workflow

#### Simple Screenshot Analysis
```bash
# User takes screenshot (Cmd+Shift+4 on Mac)
$ l8s paste dev-embed
Pasted to /tmp/claude-clipboard/clipboard.png

# In Claude Code
/paste What's wrong with this UI?
```

#### Named Files for Multiple Items
```bash
$ l8s paste dev-embed error1
Pasted to /tmp/claude-clipboard/clipboard-error1.png

$ l8s paste dev-embed error2
Pasted to /tmp/claude-clipboard/clipboard-error2.png

# In Claude Code, reference directly
"Compare /tmp/claude-clipboard/clipboard-error1.png with /tmp/claude-clipboard/clipboard-error2.png"
```

#### Text Snippet Sharing
```bash
# User copies code/text
$ l8s paste dev-embed
Pasted to /tmp/claude-clipboard/clipboard.txt

# In Claude Code
/paste Can you refactor this code?
```

## Testing Requirements

### Unit Tests
1. Test clipboard type detection on each platform
2. Test file naming logic (default vs named)
3. Test error handling for empty clipboard
4. Test SSH command construction

### Integration Tests
1. Test full paste flow with running container
2. Test file cleanup behavior for defaults
3. Test permission and ownership of created files
4. Test with various content types (PNG, JPEG, plain text, rich text)

### Manual Testing Checklist
- [ ] Paste screenshot with default name
- [ ] Paste screenshot with custom name
- [ ] Paste text with default name
- [ ] Paste text with custom name
- [ ] Paste with empty clipboard (should error)
- [ ] Paste to non-existent container (should error)
- [ ] Paste to stopped container (should error)
- [ ] Verify `/paste` hook finds correct file
- [ ] Verify old default files are cleaned up
- [ ] Verify named files persist

## Future Enhancements

1. **Additional formats**: PDF, HTML, RTF support
2. **Paste history**: Keep last N pastes with timestamps
3. **Two-way sync**: Copy from container to local clipboard
4. **Directory support**: Paste multiple files as a tar archive
5. **Size limits**: Add configurable max file size
6. **Compression**: Compress large files before transfer
7. **Progress indicator**: Show progress for large transfers
8. **Configuration**: Allow custom default directory instead of `/tmp/claude-clipboard/`

## Security Considerations

1. **File permissions**: Ensure pasted files are only readable by container user
2. **Path validation**: Prevent directory traversal in custom names
3. **Size limits**: Implement maximum file size to prevent DoS
4. **Sanitization**: Sanitize file names to prevent command injection
5. **Temporary file cleanup**: Clean up local temp files after transfer

## Dependencies

- **macOS**: osascript, pbpaste (built-in)
- **Linux**: xclip/wl-paste (may need installation)
- **Windows**: PowerShell (built-in)
- **All platforms**: SSH access to container

## Success Metrics

1. Time from screenshot to Claude analysis < 5 seconds
2. Support for 95% of common clipboard content types
3. Zero residual files left on host machine after transfer
4. Clear error messages that guide users to resolution
