# Cobra Command Grouping Architecture

Cobra provides built-in support for organizing commands into groups for better help output organization.

## Core Components

### Group Structure
```go
cobra.Group{
    ID:    "group-id",
    Title: "Group Title:"
}
```

### Group Registration
Groups must be registered on the parent command before assigning child commands:
```go
rootCmd.AddGroup(&cobra.Group{...})
```

### Command Assignment
Assign commands to groups via the GroupID field:
```go
cmd.GroupID = "group-id"
```

## Key Requirements

1. **Registration Order**: Groups MUST be defined via AddGroup() before any command sets its GroupID
2. **Error Prevention**: Setting GroupID before AddGroup() causes "Group id 'X' is not defined" error
3. **Complete Coverage**: Use AllChildCommandsHaveGroup() to verify all commands are grouped

## Helper Methods

### Built-in Commands
- `SetHelpCommandGroupId(groupID)` - Assign help command to a group
- `SetCompletionCommandGroupId(groupID)` - Assign completion command to a group

### Validation
- `ContainsGroup(groupID)` - Check if a group exists
- `AllChildCommandsHaveGroup()` - Verify all commands have groups

## Display Order

Groups appear in the help output in the order they are defined via AddGroup(), not alphabetically.

## Best Practices

1. Define all groups immediately after creating the root command
2. Use descriptive group IDs that indicate purpose
3. Keep group titles concise but clear
4. Ensure every command belongs to exactly one group
5. Test help output to verify proper grouping

## Related Files
- `cmd/l8s/main.go` - Command registration and group setup