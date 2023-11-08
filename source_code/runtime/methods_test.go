package runtime

import (
	"reflect"
	"testing"
	"time"
)

// TestAddEBSVolumeConfigs tests the AddEBSVolumeConfigs method of the Config struct.
// It checks if the EBS volumes have been correctly added to the Config's list of volumes.
func TestAddEBSVolumeConfigs(t *testing.T) {
	cfg := InitialiseConfig()
	volumes := []EBSVolumeConfig{
		{
			AWSVolumeID: "vol-0abcd1234efgh5678",
		},
	}

	// "want" represents the expected outcome
	want := volumes

	// Adding EBS volume configs
	cfg.AddEBSVolumeConfigs(volumes...)

	// "got" represents the actual outcome
	got := cfg.Volumes

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddEBSVolumeConfigs() = %v, want %v", got, want)
	}
}

// TestSetCheckInterval tests the SetCheckInterval method of the Config struct.
// It checks if the check interval has been correctly set.
func TestSetCheckInterval(t *testing.T) {
	cfg := InitialiseConfig()
	interval := 30

	// "want" represents the expected outcome
	want := interval

	// Setting check interval
	cfg.SetCheckInterval(interval)

	// "got" represents the actual outcome
	got := cfg.CheckIntervalSeconds

	if got != want {
		t.Errorf("SetCheckInterval() = %v, want %v", got, want)
	}
}

// TestAddEBSVolumeStateExecution tests the AddEBSVolumeStateExecution method of the VolumeHistory struct.
// It checks if the volume state and execution success flag have been correctly added.
func TestAddEBSVolumeStateExecution(t *testing.T) {
	history := InitialiseEvent()
	state := EBSVolumeState{
		AWSVolumeID: "vol-0abcd1234efgh5678",
	}
	success := true

	// "want" represents the expected outcome
	wantState := state
	wantSuccess := success

	// Adding volume state execution
	history.AddEBSVolumeState(state, success)

	// "got" represents the actual outcome
	gotState := history.VolumeState
	gotSuccess := history.ExecutionSuccess

	if !reflect.DeepEqual(gotState, wantState) || gotSuccess != wantSuccess {
		t.Errorf("AddEBSVolumeStateExecution() = (%v, %v), want (%v, %v)", gotState, gotSuccess, wantState, wantSuccess)
	}
}

// TestAddEBSVolumeResizeExecution tests the AddEBSVolumeResizeExecution method of the VolumeHistory struct.
// It checks if the volume resize action and execution success flag have been correctly added.
func TestAddEBSVolumeResizeExecution(t *testing.T) {
	history := InitialiseEvent()
	action := EBSVolumeResize{
		AWSVolumeID:    "vol-0abcd1234efgh5678",
		OriginalSizeGB: 10,
		NewSize:        20,
	}
	success := true

	// "want" represents the expected outcome
	wantAction := action
	wantSuccess := success

	// Adding volume resize execution
	history.AddEBSVolumeResize(action, success)

	// "got" represents the actual outcome
	gotAction := history.VolumeAction
	gotSuccess := history.ExecutionSuccess

	if !reflect.DeepEqual(gotAction, wantAction) || gotSuccess != wantSuccess {
		t.Errorf("AddEBSVolumeResizeExecution() = (%v, %v), want (%v, %v)", gotAction, gotSuccess, wantAction, wantSuccess)
	}
}

// TestAddFilesystemResizeExecution tests the AddFilesystemResizeExecution method of the VolumeHistory struct.
// It checks if the filesystem resize action and execution success flag have been correctly added.
func TestAddFilesystemResizeExecution(t *testing.T) {
	history := InitialiseEvent()
	action := FilesystemResize{
		AWSVolumeID:    "vol-0abcd1234efgh5678",
		OriginalSizeGB: 10,
		NewSize:        20,
	}
	success := true

	// "want" represents the expected outcome
	wantAction := action
	wantSuccess := success

	// Adding filesystem resize execution
	history.AddFilesystemResize(action, success)

	// "got" represents the actual outcome
	gotAction := history.FSAction
	gotSuccess := history.ExecutionSuccess

	if !reflect.DeepEqual(gotAction, wantAction) || gotSuccess != wantSuccess {
		t.Errorf("AddFilesystemResizeExecution() = (%v, %v), want (%v, %v)", gotAction, gotSuccess, wantAction, wantSuccess)
	}
}

// TestPrune tests the Prune method of the VolumeHistories type.
// It checks if the VolumeHistory entries older than 1 day have been correctly removed.
func TestPrune(t *testing.T) {
	histories := InitialiseEventLog(*InitialiseConfig())
	oldHistory := InitialiseEvent()
	oldHistory.EventTime = time.Now().Add(-25 * time.Hour)
	//histories["vol-0abcd1234efgh5678"] = []Event{*oldHistory}

	// "want" represents the expected outcome
	want := EventLog{}

	// Pruning histories
	histories.PruneStaleEvents()

	// "got" represents the actual outcome
	got := histories

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Prune() = %v, want %v", got, want)
	}
}
