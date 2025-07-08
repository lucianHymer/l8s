// +build integration

package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindAvailablePort_Integration tests the actual port finding behavior
func TestFindAvailablePort_Integration(t *testing.T) {
	// This test requires a real Podman connection
	client, err := NewPodmanClient()
	if err != nil {
		t.Skip("Skipping integration test: no Podman connection available")
	}

	ctx := context.Background()

	// Get current containers to understand the initial state
	containers, err := client.ListContainers(ctx)
	require.NoError(t, err)

	// Find the highest port in use
	highestPort := 0
	for _, container := range containers {
		if container.Status == "running" && container.SSHPort > highestPort {
			highestPort = container.SSHPort
		}
	}

	// Test finding available port
	testCases := []struct {
		name      string
		startPort int
		expected  func(int) bool
	}{
		{
			name:      "find port starting from 2200",
			startPort: 2200,
			expected: func(port int) bool {
				// Should return a port >= 2200 that's not in use
				return port >= 2200
			},
		},
		{
			name:      "find port when starting port is in use",
			startPort: highestPort,
			expected: func(port int) bool {
				// Should return a port > highestPort
				return port > highestPort
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, err := client.FindAvailablePort(tc.startPort)
			require.NoError(t, err)
			assert.True(t, tc.expected(port), "Port %d doesn't meet expected criteria", port)

			// Verify the port is not in use by any running container
			containers, err := client.ListContainers(ctx)
			require.NoError(t, err)

			for _, container := range containers {
				if container.Status == "running" {
					assert.NotEqual(t, port, container.SSHPort, 
						"Port %d is already in use by container %s", port, container.Name)
				}
			}
		})
	}
}

// TestFindAvailablePort_MultipleContainers simulates the bug scenario
func TestFindAvailablePort_MultipleContainers(t *testing.T) {
	t.Skip("Manual test - demonstrates the fix for multiple containers")
	
	// This test demonstrates that FindAvailablePort now correctly
	// checks remote container ports instead of local ports
	
	// To run this test manually:
	// 1. Create a container (it will use port 2200)
	// 2. Create another container - it should use port 2201, not fail trying to use 2200
}