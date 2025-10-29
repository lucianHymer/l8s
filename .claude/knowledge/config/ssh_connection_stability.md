# SSH Connection Stability Configuration

L8s configures SSH connections with enhanced keepalive settings for improved stability and reliability.

## Configuration Settings

### ControlPersist
**Value**: 1 hour (changed from 10 minutes)
- Multiplexed connections stay alive longer
- Reduces connection overhead for frequent operations
- Better for long development sessions

### ServerAliveInterval
**Value**: 30 seconds
- Client sends keepalive packets every 30 seconds
- Maintains connection through idle periods
- Prevents timeout on inactive connections

### ServerAliveCountMax
**Value**: 6
- Tolerates 6 failed keepalive attempts
- Total tolerance: 6 × 30s = 3 minutes
- Faster detection of dead connections vs indefinite hang

### ConnectTimeout
**Value**: 10 seconds
- Faster timeout on initial connection attempts
- Prevents long hangs on unreachable hosts
- Better user experience for connection failures

### TCPKeepAlive
**Value**: yes
- Explicit TCP-level keepalives
- Complements SSH-level keepalives
- Better detection of network issues

## Benefits

1. **Network Resilience**: Connections survive brief network hiccups
2. **Faster Recovery**: Dead connections detected in ~3 minutes vs indefinite
3. **Better Performance**: Connection multiplexing lasts longer
4. **User Experience**: Faster feedback on connection issues

## Implementation

Settings applied in `GenerateSSHConfigEntry()`:
- Used for both CA-enabled and fallback SSH configs
- Consistent behavior across all container connections
- Configured automatically during container creation

## Related Files
- `pkg/ssh/keys.go` - SSH config generation with keepalive settings
- `pkg/ssh/keys_test.go` - Tests for SSH config generation
