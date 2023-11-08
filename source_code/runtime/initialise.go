package runtime

import "time"

// InitialiseConfig initializes an empty Config struct.
// return : *Config Newly created Config.
func InitialiseConfig() *Config {
	return &Config{}
}

// InitialiseRuntime initializes an empty Runtime struct.
// return : *Runtime Newly created
func InitialiseRuntime() *Runtime {
	return &Runtime{}
}

// InitialiseEventLog creates a new EventLog map with an empty history for each volume.
// config : Config The configuration that includes the volumes to initialise the eventlog for.
// return : EventLog Initialised VolumeHistories map.
func InitialiseEventLog(config Config) EventLog {
	eventLog := make(EventLog)
	for _, volumeConfig := range config.Volumes {
		eventLog[volumeConfig.AWSVolumeID] = make([]Event, 0)
	}
	return eventLog
}

// InitialiseEvent initializes an empty Event struct.
// return : *Event Newly created Event.
func InitialiseEvent() Event {
	return Event{}
}

// InitialiseEBSVolumeState initializes an empty EBSVolumeState struct.
// return : *EBSVolumeState Newly created EBSVolumeState.
func InitialiseEBSVolumeState() EBSVolumeState {
	return EBSVolumeState{}
}

// CreateVolumeStateEvent creates an event based on a volume state action.
// volumeState : EBSVolumeState state of the volume
// success : bool indicates if the action was successful
// errorCount : int current error count
// returns : Event created event
func CreateVolumeStateEvent(volumeState EBSVolumeState, success bool) Event {
	event := InitialiseEvent()
	event.EventTime = time.Now()
	event.VolumeState = volumeState
	event.ExecutionSuccess = success
	return event
}

// CreateVolumeResizeActionEvent creates an event based on a volume action.
// volumeAction : EBSVolumeResize action taken on the volume
// success : bool indicates if the action was successful
// errorCount : int current error count
// returns : Event created event
func CreateVolumeResizeActionEvent(volumeAction EBSVolumeResize, success bool) Event {
	event := InitialiseEvent()
	event.EventTime = time.Now()
	event.VolumeAction = volumeAction
	event.ExecutionSuccess = success
	return event
}

// CreateFSActionEvent creates an event based on a file system action.
// fsAction : FilesystemResize action taken on the file system
// success : bool indicates if the action was successful
// errorCount : int current error count
// returns : Event created event
func CreateFSActionEvent(fsAction FilesystemResize, success bool) Event {
	event := InitialiseEvent()
	event.EventTime = time.Now()
	event.FSAction = fsAction
	event.ExecutionSuccess = success
	return event
}
