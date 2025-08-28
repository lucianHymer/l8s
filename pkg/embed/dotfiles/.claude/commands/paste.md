---
allowed-tools: Bash(test:/tmp/claude-clipboard/*), Bash(echo:*)
---
!if [ -f /tmp/claude-clipboard/clipboard.png ]; then echo "Screenshot pasted to /tmp/claude-clipboard/clipboard.png"; elif [ -f /tmp/claude-clipboard/clipboard.txt ]; then echo "Text pasted to /tmp/claude-clipboard/clipboard.txt"; else echo "No pasted content found"; fi

Please analyze the pasted content mentioned above.

$ARGUMENTS