package filesystem

import (
	"testing"
)

// TestGetLocalMountPoint tests the GetLocalMountPoint function.
func TestGetLocalMountPoint(t *testing.T) {
	testCases := []struct {
		deviceName string
		expected   string
		wantErr    bool
	}{
		{
			deviceName: "/dev/sda1",
			expected:   "/mnt/xvda1",
			wantErr:    false,
		},
		{
			deviceName: "/dev/sdb",
			expected:   "/mnt/xvdb",
			wantErr:    false,
		},
	}

	for _, tc := range testCases {
		result, err := GetLocalMountPoint(tc.deviceName)
		if result != tc.expected || err != nil {
			t.Errorf("Device name %s: Expected %s, got %s", tc.deviceName, tc.expected, result)
		}
	}
}

// TODO: add additional tests - requires mocking external calls
