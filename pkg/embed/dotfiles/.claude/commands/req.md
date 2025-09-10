---
allowed-tools: Bash(test:*), Bash(cat:*), Bash(echo:*), Bash(cut:*)
argument-hint: <file-path> [additional clarification]
description: Read and analyze requirements from a specified file
---
!if [ -z "$1" ]; then echo "Error: Please specify a file path as the first argument"; exit 1; fi
!if [ ! -f "$1" ]; then echo "Error: File '$1' not found"; exit 1; fi
!echo "=== Reading requirements from: $1 ==="
!cat "$1"
!echo "=== End of requirements file ==="

Can you take a look at $1 and the related code files? Ask any clarifying questions, then write a haiku (no abbreviations, strict haiku form) which you think demonstrates your deep and full understanding of the request. Finally, explain your implementation plan.

!context=$(echo "$ARGUMENTS" | cut -d' ' -f2-); if [ -n "$context" ]; then echo ""; echo "$context"; fi