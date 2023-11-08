package configutil

import (
	"ebs-monitor/runtime"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// TestGetConfigFromFile : a test function for GetConfigFromFile.
func TestGetConfigFromFile(t *testing.T) {
	viper.SetConfigType("yaml")

	tests := []struct {
		name              string
		configFile        string
		wantVolumes       []runtime.EBSVolumeConfig
		wantCheckInverval int
		wantErr           bool
	}{
		{
			name:       "Valid config file",
			configFile: "config_test.yaml",
			wantVolumes: []runtime.EBSVolumeConfig{
				{
					AWSVolumeID:          "vol-0abcd1234efgh5678",
					AWSDeviceName:        "/dev/sdf",
					AWSRegion:            "ap-southeast-2",
					IncrementSizeGB:      10,
					IncrementSizePercent: 20,
					ResizeThreshold:      30,
				},
				{
					AWSVolumeID:          "vol-0efgh5678abcd1234",
					AWSDeviceName:        "/dev/sdg",
					AWSRegion:            "ap-southeast-2",
					IncrementSizeGB:      20,
					IncrementSizePercent: 30,
					ResizeThreshold:      40,
				},
			},
			wantCheckInverval: 30,
			wantErr:           false,
		},
		{
			name:              "Missing config file",
			configFile:        "invalid_config_test.yaml",
			wantVolumes:       nil,
			wantCheckInverval: 0,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVolumes, gotCheckInterval, err := GetConfigFromFile(tt.configFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotCheckInterval != tt.wantCheckInverval {
				t.Errorf("GetConfigFromFile() check interval = %v, wantCheckInterval %v", gotCheckInterval, tt.wantCheckInverval)
			}

			if !reflect.DeepEqual(gotVolumes, tt.wantVolumes) {
				t.Errorf("GetConfigFromFile() = %v, want %v", gotVolumes, tt.wantVolumes)
			}
		})
	}

}

// TestValidatePositiveInt : a test function for validatePositiveInt.
func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name    string
		num     int
		wantErr bool
	}{
		{
			name:    "Valid number",
			num:     10,
			wantErr: false,
		},
		{
			name:    "Invalid number",
			num:     -5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePositiveInt(tt.num)

			if (err != nil) != tt.wantErr {
				t.Errorf("validatePositiveInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestCheckMinimumFields tests the checkMinimumFields function
func TestCheckMinimumFields(t *testing.T) {
	tests := []struct {
		name     string
		volume   runtime.EBSVolumeConfig
		expected bool
	}{
		{
			name: "valid volume configuration",
			volume: runtime.EBSVolumeConfig{
				AWSVolumeID:          "vol-0abcd1234efgh5678",
				AWSDeviceName:        "/dev/sdf",
				AWSRegion:            "us-east-1",
				IncrementSizeGB:      10,
				IncrementSizePercent: 20,
				ResizeThreshold:      80,
			},
			expected: true,
		},
		{
			name: "invalid volume configuration",
			volume: runtime.EBSVolumeConfig{
				AWSVolumeID:          "",
				AWSDeviceName:        "",
				AWSRegion:            "",
				IncrementSizeGB:      0,
				IncrementSizePercent: 0,
				ResizeThreshold:      0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkMinimumFields(tt.volume); got != tt.expected {
				t.Errorf("checkMinimumFields() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TODO Add additional tests for external calling functions. Requires gomock.
