package runtime

import (
	"reflect"
	"testing"
	"time"
)

// TestInitialiseConfig tests the InitialiseConfig function using table driven tests.
func TestInitialiseConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "Initialise empty config",
			want: &Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialiseConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitialiseConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInitialiseRuntime tests the InitialiseRuntime function using table driven tests.
func TestInitialiseRuntime(t *testing.T) {
	tests := []struct {
		name string
		want *Runtime
	}{
		{
			name: "Initialise empty runtime",
			want: &Runtime{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialiseRuntime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitialiseRuntime() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInitialiseEvent tests the InitialiseEvent function using table driven tests.
func TestInitialiseEvent(t *testing.T) {
	tests := []struct {
		name string
		want Event
	}{
		{
			name: "Initialise empty event",
			want: Event{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialiseEvent(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitialiseEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInitialiseEventLog tests the InitialiseEventLog function using table driven tests.
func TestInitialiseEventLog(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want EventLog
	}{
		{
			name: "Initialise EventLog with no volumes",
			cfg:  Config{},
			want: make(EventLog),
		},
		{
			name: "Initialise EventLog with one volume",
			cfg: Config{
				Volumes: []EBSVolumeConfig{
					{
						AWSVolumeID: "vol-0abcd1234efgh5678",
					},
				},
			},
			want: EventLog{
				"vol-0abcd1234efgh5678": []Event{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitialiseEventLog(tt.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitialiseEventLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateVolumeStateEvent(t *testing.T) {
	// Test cases for CreateVolumeStateEvent function
	testCases := []struct {
		volumeState EBSVolumeState
		success     bool
		wantErr     bool
	}{
		{
			volumeState: EBSVolumeState{
				AWSVolumeID:     "vol-123",
				AWSDeviceName:   "/dev/sda",
				LocalMountPoint: "/mnt/volume",
				AWSDeviceSizeGB: 100,
				LocalDiskSizeGB: 100,
				UsedSpaceGB:     50,
			},
			success: true,
			wantErr: false,
		},
		{
			volumeState: EBSVolumeState{
				AWSVolumeID:     "",
				AWSDeviceName:   "",
				LocalMountPoint: "",
				AWSDeviceSizeGB: 0,
				LocalDiskSizeGB: 0,
				UsedSpaceGB:     0,
			},
			success: false,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		event := CreateVolumeStateEvent(tc.volumeState, tc.success)
		if event.EventTime.IsZero() {
			t.Error("Event time should not be zero")
		}
		if !compareVolumeStates(event.VolumeState, tc.volumeState) {
			t.Errorf("Expected volume state %v, but got %v", tc.volumeState, event.VolumeState)
		}
		if !compareVolumeActions(event.VolumeAction, EBSVolumeResize{}) {
			t.Errorf("Expected empty volume action, but got %v", event.VolumeAction)
		}
		if !compareFSActions(event.FSAction, FilesystemResize{}) {
			t.Errorf("Expected empty filesystem action, but got %v", event.FSAction)
		}
		if event.ExecutionSuccess != !tc.wantErr {
			t.Errorf("Expected execution success %v, but got %v", !tc.wantErr, event.ExecutionSuccess)
		}
	}
}

func TestCreateVolumeResizeActionEvent(t *testing.T) {
	// Test cases for CreateVolumeResizeActionEvent function
	testCases := []struct {
		volumeAction EBSVolumeResize
		success      bool
	}{
		{
			volumeAction: EBSVolumeResize{
				StartTime:      time.Now(),
				AWSVolumeID:    "vol-123",
				AWSDeviceName:  "/dev/sda",
				AWSRegion:      "us-west-2",
				OriginalSizeGB: 100,
				NewSize:        200,
			},
			success: true,
		},
		{
			volumeAction: EBSVolumeResize{
				StartTime:      time.Now(),
				AWSVolumeID:    "vol-456",
				AWSDeviceName:  "/dev/sdb",
				AWSRegion:      "eu-west-1",
				OriginalSizeGB: 200,
				NewSize:        300,
			},
			success: false,
		},
	}

	for _, tc := range testCases {
		event := CreateVolumeResizeActionEvent(tc.volumeAction, tc.success)
		if event.EventTime.IsZero() {
			t.Error("Event time should not be zero")
		}
		if !compareVolumeActions(event.VolumeAction, tc.volumeAction) {
			t.Errorf("Expected volume action %v, but got %v", tc.volumeAction, event.VolumeAction)
		}
		if !compareFSActions(event.FSAction, FilesystemResize{}) {
			t.Errorf("Expected empty filesystem action, but got %v", event.FSAction)
		}
		if event.ExecutionSuccess != tc.success {
			t.Errorf("Expected execution success %v, but got %v", tc.success, event.ExecutionSuccess)
		}
	}
}

func TestCreateFSActionEvent(t *testing.T) {
	// Test cases for CreateFSActionEvent function
	testCases := []struct {
		fsAction FilesystemResize
		success  bool
	}{
		{
			fsAction: FilesystemResize{
				StartTime:       time.Now(),
				AWSVolumeID:     "vol-123",
				AWSDeviceName:   "/dev/sda",
				LocalMountPoint: "/mnt/volume",
				AWSVolumeSize:   100,
				OriginalSizeGB:  50,
				NewSize:         100,
			},
			success: true,
		},
		{
			fsAction: FilesystemResize{
				StartTime:       time.Now(),
				AWSVolumeID:     "vol-456",
				AWSDeviceName:   "/dev/sdb",
				LocalMountPoint: "/mnt/data",
				AWSVolumeSize:   200,
				OriginalSizeGB:  100,
				NewSize:         200,
			},
			success: false,
		},
	}

	for _, tc := range testCases {
		event := CreateFSActionEvent(tc.fsAction, tc.success)
		if event.EventTime.IsZero() {
			t.Error("Event time should not be zero")
		}
		if !compareFSActions(event.FSAction, tc.fsAction) {
			t.Errorf("Expected filesystem action %v, but got %v", tc.fsAction, event.FSAction)
		}
		if event.VolumeState != (EBSVolumeState{}) {
			t.Errorf("Expected empty volume state, but got %v", event.VolumeState)
		}
		if event.VolumeAction != (EBSVolumeResize{}) {
			t.Errorf("Expected empty volume action, but got %v", event.VolumeAction)
		}
		if event.ExecutionSuccess != tc.success {
			t.Errorf("Expected execution success %v, but got %v", tc.success, event.ExecutionSuccess)
		}
	}
}

// Helper function to compare VolumeAction values
func compareVolumeActions(action1, action2 EBSVolumeResize) bool {
	return action1.StartTime.Equal(action2.StartTime) &&
		action1.AWSVolumeID == action2.AWSVolumeID &&
		action1.AWSDeviceName == action2.AWSDeviceName &&
		action1.AWSRegion == action2.AWSRegion &&
		action1.OriginalSizeGB == action2.OriginalSizeGB &&
		action1.NewSize == action2.NewSize
}

// Helper function to compare FSAction values
func compareFSActions(action1, action2 FilesystemResize) bool {
	return action1.StartTime.Equal(action2.StartTime) &&
		action1.AWSVolumeID == action2.AWSVolumeID &&
		action1.AWSDeviceName == action2.AWSDeviceName &&
		action1.LocalMountPoint == action2.LocalMountPoint &&
		action1.AWSVolumeSize == action2.AWSVolumeSize &&
		action1.OriginalSizeGB == action2.OriginalSizeGB &&
		action1.NewSize == action2.NewSize
}

// Helper function to compare VolumeState values
func compareVolumeStates(state1, state2 EBSVolumeState) bool {
	return state1.AWSVolumeID == state2.AWSVolumeID &&
		state1.AWSDeviceName == state2.AWSDeviceName &&
		state1.LocalMountPoint == state2.LocalMountPoint &&
		state1.AWSDeviceSizeGB == state2.AWSDeviceSizeGB &&
		state1.LocalDiskSizeGB == state2.LocalDiskSizeGB &&
		state1.UsedSpaceGB == state2.UsedSpaceGB
}
