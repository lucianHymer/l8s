# Cobra Command Structure Pattern

Commands in L8s follow a consistent structure using the Cobra framework.

## Command Definition Structure

Created in factory methods with `cobra.Command`:
- **Use/Short/Long**: Command descriptions and documentation
- **Args validation**: `cobra.ExactArgs(1)`, `cobra.MinimumNArgs(2)`, etc.
- **RunE function**: Execution with error handling support
- **Flags**: Defined using `cmd.Flags().StringVar()`, `cmd.Flags().BoolVar()`

## Handler Pattern

Execution is delegated to handler methods:
1. `LazyCommandFactory.XXXCmd()` creates command and ensures initialization
2. `CommandFactory.runXXX()` methods contain actual logic
3. Error handling occurs in RunE functions, not in handlers
4. Consistent error propagation throughout

## Flag Handling

Flags are retrieved in handlers using:
- `cmd.Flags().GetString()` / `GetBool()` methods
- Validation happens in handlers, not command definition
- Mutually exclusive flags validated explicitly
- Default values set during flag definition

## Context Usage

All handlers accept `context.Context` for operations:
- Passed to container manager and other dependencies
- Enables timeout and cancellation support
- Consistent context propagation through call chain

## Example Pattern

```go
func (f *LazyCommandFactory) PasteCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "paste <container> [name]",
        Short: "Paste clipboard to container",
        Args:  cobra.RangeArgs(1, 2),
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := f.ensureInitialized(); err != nil {
                return err
            }
            return f.factory.runPaste(cmd.Context(), args)
        },
    }
    return cmd
}
```

## Related Files
- `pkg/cli/factory_lazy.go` - Command definitions
- `pkg/cli/handlers.go` - Handler implementations