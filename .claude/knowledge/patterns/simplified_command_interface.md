# Simplified Command Interface Pattern

When designing CLI commands, prefer optional arguments over subcommands for simple list/action patterns.

## Pattern Overview

For commands that have a "list" operation and a "take action" operation, consider making the argument optional rather than creating subcommands. When no argument is provided, show the list; when an argument is provided, take the action.

## Example: Team Command

The team command was simplified from:
- `l8s team list` (subcommand for listing)
- `l8s team <name>` (join/create session)

To:
- `l8s team` (no args → list)
- `l8s team <name>` (join/create session)

## Implementation Pattern

1. Change `Args` from `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
2. Add conditional logic in `RunE`: check `len(args)` to determine behavior
3. Update help text to reflect optional argument: `[session-name]` instead of `<session-name>`
4. Remove subcommands that only serve list/show purposes

```go
// Example implementation
cmd := &cobra.Command{
    Use:   "team [session-name]",
    Short: "Join or create a team session, or list active sessions",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) == 0 {
            // List behavior
            return listSessions()
        }
        // Join/create behavior
        return joinSession(args[0])
    },
}
```

## Benefits

1. **Intuitive UX**: Matches familiar tool patterns like tmux/screen
2. **Reduced cognitive load**: Fewer commands to remember
3. **Discoverability**: Users get a list if they don't specify what they want
4. **Fewer keystrokes**: No need to type "list" subcommand

## When to Use This Pattern

Use this pattern when:
- You have a simple list/action pair of operations
- The list operation takes no additional arguments
- The action operation takes exactly one argument
- Showing the list by default helps users discover available options

Avoid this pattern when:
- Multiple subcommands exist beyond just "list"
- The action requires multiple arguments or flags
- The list operation needs filtering options

## Related Files
- `pkg/cli/factory_lazy.go` - Command definitions
- `pkg/embed/dotfiles/.local/bin/team` - Example script using this pattern
