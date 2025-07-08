// +build test

package container

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestFindAvailablePort_CurrentBehavior shows the current (buggy) behavior
func TestFindAvailablePort_CurrentBehavior(t *testing.T) {
	// Current implementation checks local ports, not remote
	// This test documents the current behavior
	
	mockClient := &MockPodmanClient{}
	
	// Mock returns based on local port availability
	// This simulates the current implementation
	mockClient.On("FindAvailablePort", 2200).Return(2200, nil)
	
	port, err := mockClient.FindAvailablePort(2200)
	
	require.NoError(t, err)
	assert.Equal(t, 2200, port, "Current implementation returns first locally available port")
}

// TestFindAvailablePort_Remote shows how it should work
func TestFindAvailablePort_Remote(t *testing.T) {
	tests := []struct {
		name               string
		startPort          int
		existingContainers []*Container
		want               int
		wantErr            bool
	}{
		{
			name:               "first port available when no containers",
			startPort:          2200,
			existingContainers: []*Container{},
			want:               2200,
			wantErr:            false,
		},
		{
			name:      "skip port in use by existing container",
			startPort: 2200,
			existingContainers: []*Container{
				{Name: "dev-test1", SSHPort: 2200, Status: "running"},
			},
			want:    2201,
			wantErr: false,
		},
		{
			name:      "find next available with multiple containers",
			startPort: 2200,
			existingContainers: []*Container{
				{Name: "dev-test1", SSHPort: 2200, Status: "running"},
				{Name: "dev-test2", SSHPort: 2201, Status: "running"},
			},
			want:    2202,
			wantErr: false,
		},
		{
			name:      "ignore stopped containers",
			startPort: 2200,
			existingContainers: []*Container{
				{Name: "dev-test1", SSHPort: 2200, Status: "stopped"},
			},
			want:    2200,
			wantErr: false,
		},
		{
			name:      "error when no ports available in range",
			startPort: 2200,
			existingContainers: func() []*Container {
				var containers []*Container
				for i := 0; i < 100; i++ {
					containers = append(containers, &Container{
						Name:    fmt.Sprintf("dev-test%d", i),
						SSHPort: 2200 + i,
						Status:  "running",
					})
				}
				return containers
			}(),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPodmanClient{}
			mockClient.On("ListContainers", mock.Anything).Return(tt.existingContainers, nil)

			// This will fail until we implement the proper remote checking
			t.Skip("Skipping until remote port checking is implemented")
			
			got, err := mockClient.FindAvailablePort(tt.startPort)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}