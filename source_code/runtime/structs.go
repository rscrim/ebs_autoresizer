package runtime

import "time"

// Runtime represents the runtime state of the application, including the loaded configuration and
// a debug mode toggle for verbose output.
type Runtime struct {
	Configuration Config // Configuration loaded from the config.yaml file.
	DebugMode     bool   // Indicates if the application is running in debug mode.
}

// Config represents the runtime configuration of the system.
// It includes the list of EBS volumes to be monitored and the frequency of checks.
type Config struct {
	Volumes              []EBSVolumeConfig // List of EBS volumes to be managed.
	CheckIntervalSeconds int               `yaml:"checkIntervalSeconds"` // Frequency of checking volume state in seconds.
}

// EBSVolumeConfig represents the configuration for an EBS volume.
type EBSVolumeConfig struct {
	AWSVolumeID          string `yaml:"awsVolumeID"`          // Identifier for the EBS volume.
	AWSDeviceName        string `yaml:"awsDeviceName"`        // Name of the EBS device.
	AWSRegion            string `yaml:"awsRegion"`            // AWS region where the EBS volume is located.
	IncrementSizeGB      int    `yaml:"incrementSizeGB"`      // Size to increase volume by (in GB), when required.
	IncrementSizePercent int    `yaml:"incrementSizePercent"` // Percentage to increase volume size, when required.
	ResizeThreshold      int    `yaml:"resizeThreshold"`      // Threshold percentage at which to resize the volume.
}

// EventLog represents a map of volume histories.
// It maps AWS Volume IDs to slices of VolumeHistory.
type EventLog map[string][]Event

// Event represents the history of actions taken on a specific EBS volume.
// It includes timestamps, volume states, actions, and success flags.
type Event struct {
	EventTime        time.Time        // Time of the event.
	VolumeState      EBSVolumeState   // Snapshot of EBS volume at the time of the event.
	VolumeAction     EBSVolumeResize  // Resize action taken on the EBS volume.
	FSAction         FilesystemResize // Filesystem resize action.
	ExecutionSuccess bool             // Indicates if the action executed successfully.
}

// EBSVolumeState represents a snapshot of an EBS volume at a point in time.
// It includes various size and space measurements, as well as identifiers.
type EBSVolumeState struct {
	AWSVolumeID     string  // Identifier for the EBS volume.
	AWSDeviceName   string  // Name of the EBS device.
	LocalMountPoint string  // Local device name where the EBS volume is attached.
	AWSDeviceSizeGB float64 // Size of the EBS volume in gigabytes.
	LocalDiskSizeGB float64 // Size of the local disk in gigabytes.
	UsedSpaceGB     float64 // Amount of disk space used, in gigabytes.
}

// EBSVolumeResize represents a resize action on an EBS volume.
// It includes timestamps, identifiers, and the original and new sizes of the volume.
type EBSVolumeResize struct {
	StartTime      time.Time // Time when resize API request was sent.
	AWSVolumeID    string    // Identifier for the EBS volume.
	AWSDeviceName  string    // Name of the EBS device.
	AWSRegion      string    // AWS region where the EBS volume is located.
	OriginalSizeGB float64   // Original size of the EBS volume, in gigabytes.
	NewSize        float64   // New size of the EBS volume, in gigabytes.
}

// FilesystemResize represents a resize action on the local filesystem.
// It includes timestamps, identifiers, and the original and new sizes of the filesystem.
type FilesystemResize struct {
	StartTime       time.Time // Time when filesystem resize command was run.
	AWSVolumeID     string    // Identifier for the EBS volume.
	AWSDeviceName   string    // Name of the EBS device.
	LocalMountPoint string    // Local device name where the EBS volume is attached.
	AWSVolumeSize   float64   // Current size of the EBS volume, in gigabytes.
	OriginalSizeGB  float64   // Original size of the filesystem, in gigabytes.
	NewSize         float64   // New size of the filesystem, in gigabytes.
}
