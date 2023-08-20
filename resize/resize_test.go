package resize

import (
	"ebs-monitor/runtime"
	"testing"
)

// TODO: add tests - requires mocking external calls

func TestCalculateNewSize(t *testing.T) {
	tests := []struct {
		name        string
		config      runtime.EBSVolumeConfig
		currentSize int64
		expected    int64
	}{
		{
			name:        "normal case with IncrementSizeGB",
			config:      runtime.EBSVolumeConfig{IncrementSizeGB: 5},
			currentSize: 10,
			expected:    15,
		},
		{
			name:        "normal case with IncrementSizePercent",
			config:      runtime.EBSVolumeConfig{IncrementSizePercent: 20},
			currentSize: 100,
			expected:    120,
		},
		{
			name:        "normal case with default increment",
			config:      runtime.EBSVolumeConfig{},
			currentSize: 20,
			expected:    30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateNewSize(tt.config, tt.currentSize)
			if got != tt.expected {
				t.Errorf("calculateNewSize() = %v, want %v", got, tt.expected)
			}
		})
	}
}
