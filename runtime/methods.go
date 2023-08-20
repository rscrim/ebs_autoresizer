package runtime

import (
	"ebs-monitor/logger"
	"fmt"
	"time"
)

// Initialise logger
var l = logger.NewLogger()

/*
-------------------------
Methods for Runtime Struct
-------------------------
*/
// SetConfiguration sets the configuration for the Runtime.
// cfg : Config The configuration to be set.
func (runtime *Runtime) SetConfiguration(cfg Config) {
	runtime.Configuration = cfg
}

// SetDebugMode sets the debug mode for the Runtime.
// debugMode : bool The debug mode to be set.
func (runtime *Runtime) SetDebugMode(debugMode bool) {
	runtime.DebugMode = debugMode
}

/*
-------------------------
Methods for Config Struct
-------------------------
*/

// AddEBSVolumeConfigs adds one or more EBS volumes to the Config's list of volumes.
// volumes : ...EBSVolumeConfig Volumes to be added.
func (cfg *Config) AddEBSVolumeConfigs(volumes ...EBSVolumeConfig) {
	cfg.Volumes = append(cfg.Volumes, volumes...)
}

// SetCheckInterval sets the check interval for the Config.
// interval : time.Duration Check interval to be set.
func (cfg *Config) SetCheckInterval(interval int) {
	cfg.CheckIntervalSeconds = interval
}

/*
-------------------------
Methods for Event Struct
-------------------------
*/

// SetEventTime sets the time of the event.
func (e *Event) SetEventTime(t time.Time) {
	e.EventTime = t
}

// SetVolumeState sets the snapshot of EBS volume at the time of the event.
func (e *Event) SetVolumeState(vs EBSVolumeState) {
	e.VolumeState = vs
}

// SetVolumeAction sets the resize action taken on the EBS volume.
func (e *Event) SetVolumeAction(va EBSVolumeResize) {
	e.VolumeAction = va
}

// SetFSAction sets the filesystem resize action.
func (e *Event) SetFSAction(fsa FilesystemResize) {
	e.FSAction = fsa
}

// SetExecutionSuccess sets the flag indicating if the action executed successfully.
func (e *Event) SetExecutionSuccess(es bool) {
	e.ExecutionSuccess = es
}

/*
-------------------------
Methods for EventLog type (map[string][]VolumeHistory)
-------------------------
*/
// AddEBSVolumeState adds a volume state and execution success flag to a VolumeHistory.
// volumeState : EBSVolumeState Volume state to be added.
// executionSuccess : bool Success flag to be added.
func (history *Event) AddEBSVolumeState(volumeState EBSVolumeState, executionSuccess bool) {
	history.VolumeState = volumeState
	history.ExecutionSuccess = executionSuccess
}

// AddEBSVolumeResize adds a volume resize action and execution success flag to a VolumeHistory.
// volumeAction : EBSVolumeResize Volume resize action to be added.
// executionSuccess : bool Success flag to be added.
func (history *Event) AddEBSVolumeResize(volumeAction EBSVolumeResize, executionSuccess bool) {
	history.VolumeAction = volumeAction
	history.ExecutionSuccess = executionSuccess
}

// AddFilesystemResize adds a filesystem resize action and execution success flag to a VolumeHistory.
// fsAction : FilesystemResize Filesystem resize action to be added.
// executionSuccess : bool Success flag to be added.
func (history *Event) AddFilesystemResize(fsAction FilesystemResize, executionSuccess bool) {
	history.FSAction = fsAction
	history.ExecutionSuccess = executionSuccess
}

// AddEvent adds an event to the event log for a specific volume, if it's not a duplicate, and logs it.
// volumeID : string - The AWS Volume ID of the volume the event is associated with.
// event : Event - The event to be added to the log.
// logger : *logger.Logger - The logger to log the event.
func (eventLog EventLog) AddEvent(volumeID string, event Event) {
	// Extracts existing events from event log
	existingEvents, exists := eventLog[volumeID]

	// Checks for event duplication
	if exists {
		for _, existingEvent := range existingEvents {
			if existingEvent.EventTime == event.EventTime && existingEvent.ExecutionSuccess == event.ExecutionSuccess {
				// The event is a duplicate, return without adding it
				return
			}
		}
	}

	eventLog[volumeID] = append(existingEvents, event)

	// Log the event using the logger
	message := fmt.Sprintf("Event for volume %s: VolumeAction = %s, FSAction = %s, Success = %v", volumeID, event.VolumeAction.AWSDeviceName, event.FSAction.AWSDeviceName, event.ExecutionSuccess)
	fields := map[string]interface{}{
		"AWSVolumeID":      volumeID,
		"EventTime":        event.EventTime,
		"VolumeState":      event.VolumeState.AWSDeviceSizeGB,
		"VolumeAction":     event.VolumeAction.AWSDeviceName,
		"FSAction":         event.FSAction.AWSDeviceName,
		"ExecutionSuccess": event.ExecutionSuccess,
	}

	failedAction := ""
	if event.VolumeState.AWSDeviceSizeGB <= 0 {
		failedAction = "Get volume state"
	} else if event.VolumeAction.AWSDeviceName != "" {
		failedAction = "Perform AWS device resize"
	} else if event.FSAction.AWSDeviceName != "" {
		failedAction = "Resize the filesystem"
	}

	if event.ExecutionSuccess {
		l.Log(logger.LogDebug, message, fields)
	} else {
		failureMessage := fmt.Sprintf("Action failed: %s. Failed to: %s", message, failedAction)
		l.Log(logger.LogError, failureMessage, fields)
	}

}

// Equals checks if the calling Event is the same as the provided Event.
// otherEvent : Event - The Event to compare with the calling Event.
// returns : bool - True if the Events are the same, otherwise false.
func (e Event) Equals(otherEvent Event) bool {
	// Here we assume two events are considered "the same" if they share the same EventTime,
	// VolumeState, and ExecutionSuccess values. Adjust this logic if your definition of
	// "same" is different.
	return e.EventTime == otherEvent.EventTime && e.VolumeState == otherEvent.VolumeState && e.ExecutionSuccess == otherEvent.ExecutionSuccess
}

// PruneStaleEvents removes all VolumeHistory entries older than 1 day from the VolumeHistories.
func (histories EventLog) PruneStaleEvents() {
	oneDayAgo := time.Now().Add(-24 * time.Hour)

	for volumeID, volumeHistories := range histories {
		var prunedVolumeHistories []Event
		for _, history := range volumeHistories {
			if history.EventTime.After(oneDayAgo) {
				prunedVolumeHistories = append(prunedVolumeHistories, history)
			}
		}
		histories[volumeID] = prunedVolumeHistories
	}
}
