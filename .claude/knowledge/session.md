### [17:51] [feature] Slash command for reading requirements
**Details**: Need to create a /req slash command that:
1. Accepts a file path as the first argument ($1)
2. Optionally accepts additional clarification text as remaining arguments ($ARGUMENTS)
3. Should read the specified requirements file and provide context to Claude
4. Will be stored in pkg/embed/dotfiles/.claude/commands/req.md
5. Should follow the pattern of existing commands like paste.md with frontmatter for allowed-tools
6. The command should help Claude understand project requirements by reading specific requirement files
**Files**: pkg/embed/dotfiles/.claude/commands/req.md
---

### [17:54] [gotcha] Claude Code @ syntax for file references in slash commands
**Details**: The @ syntax in Claude Code slash commands directly includes file contents in the prompt. For example, @path/to/file.txt will read and include that file's contents. This means we cannot use @$1 to dynamically reference a file path from an argument - the @ syntax expects a literal file path at command definition time, not runtime variable substitution. To read a file dynamically based on an argument, we need to use bash commands with the ! prefix instead.
**Files**: pkg/embed/dotfiles/.claude/commands/req.md
---

