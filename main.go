package main

import (
	"ebs-monitor/aws"
	"ebs-monitor/configutil"
	"ebs-monitor/logger"
	"ebs-monitor/monitor"
	"ebs-monitor/resize"
	"ebs-monitor/runtime"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Initialise logger
var l = logger.NewLogger()

// How many consecutive errors before a volume is removed from monitoring
const errorThreshold = 5

// rootCmd : The root command for the EBS monitor CLI
var rootCmd = &cobra.Command{
	Use:   "ebs-monitor",
	Short: "EBS Monitor is a tool to monitor and resize attached AWS EBS volumes.",
	Long:  `An Ubuntu CLI tool to monitor and automatically resize AWS EBS volumes based on a supplied config.yaml file.`,
	Run:   run,
}

var (
	// configFile : string The path to the configuration file
	configFile string
	// debugMode : bool A flag indicating whether the application should run in debug mode and extra output sent to stdout.
	debugMode bool
)

// init : Initializes the root command
func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Run in debug mode")
}

// run : The function that runs the EBS monitor
// cmd : *cobra.Command The root command
// args : []string The arguments passed to the root command
func run(cmd *cobra.Command, args []string) {
	// Check if the filepath argument is provided
	DebugPrint(debugMode, "Running command...")
	if configFile == "" {
		l.Log(logger.LogError, "Config file path is missing", nil)
		os.Exit(1)
	}

	// Initialise core structs
	appRuntime, appConfig := InitialiseApp()

	// Load config from file
	volumes, checkIntervalSeconds, err := LoadConfig(configFile)
	if err != nil {
		l.Log(logger.LogError, "Failed to load config", map[string]interface{}{
			"error":      err,
			"configFile": configFile,
		})
		os.Exit(1)
	}

	// Initialise Runtime with config and debug mode set to true
	DebugPrint(debugMode, "Initializing core structs...")
	DebugPrint(debugMode, "Loading config from file...")
	appConfig.AddEBSVolumeConfigs(volumes...)
	appConfig.SetCheckInterval(checkIntervalSeconds)
	appRuntime.SetConfiguration(*appConfig)
	appRuntime.SetDebugMode(debugMode)
	// Set logger debug mode
	if debugMode {
		l.SetDebugMode(debugMode)
	}

	// Initialise history map for volume actions
	eventLog := runtime.InitialiseEventLog(*appConfig)
	errorLog := make(map[string]int)

	// Infinite loop until no volumes left to monitor
	for {
		DebugPrint(debugMode, "Running main monitoring loop...")
		// Check if there are volumes left to monitor
		if len(appRuntime.Configuration.Volumes) == 0 {
			l.Log(logger.LogError, "No more volumes to monitor", nil)
			os.Exit(1)
		}

		// If debug mode is enabled, print runtime state
		if debugMode {
			DebugPrint(debugMode, strings.Repeat("-", 20))
			DebugPrint(debugMode, "     RUN TIME OUTPUT     ")
			DebugPrint(debugMode, strings.Repeat("-", 20))
			DumpRuntime(appConfig, eventLog, errorLog)
			DebugPrint(debugMode, strings.Repeat("-", 20))
		}

		// Iterate through all volumes in runtime config
		for index := 0; index < len(appRuntime.Configuration.Volumes); {
			DebugPrint(debugMode, fmt.Sprintf("Checking volume at index %d", index))

			// Get volumeID of current one to check
			volume := appRuntime.Configuration.Volumes[index]

			// Get current volume state & handle any errors in this process
			volumeState, err := monitor.GetVolumeState(volume, &eventLog)
			if err != nil {
				errorLog[volume.AWSVolumeID]++
				DebugPrint(debugMode, "Encountered error when getting volume state, increasing error log count...")
				DebugPrint(debugMode, fmt.Sprintf("error: %v", err))
			} else {
				errorLog[volume.AWSVolumeID] = 0 // Reset the error count on successful operation
				DebugPrint(debugMode, "Volume state retrieved successfully, resetting error log count...")
			}

			// Prints runtime state if debugmode is true
			if debugMode {
				PrintStructFields(volumeState, "")
			}

			if err != nil {
				// Create an event based on the volume state
				event := runtime.CreateVolumeStateEvent(volumeState, false)

				// Add the event to the log
				eventLog.AddEvent(volume.AWSVolumeID, event)

				// If error threshold has exceeded errorThreshold, drop the volume and log fatal error.
				if errorLog[volume.AWSVolumeID] >= errorThreshold {
					// Remove volume from the list
					appRuntime.Configuration.Volumes = append(appRuntime.Configuration.Volumes[:index], appRuntime.Configuration.Volumes[index+1:]...)
					l.Log(logger.LogError, "A disk has been removed due to recurrent errors", map[string]interface{}{
						"VolumeID":    volume.AWSVolumeID,
						"Error Count": errorLog[volume.AWSVolumeID],
					})
					continue
				}

			} else {
				// Create an event based on the volume state
				event := runtime.CreateVolumeStateEvent(volumeState, true)

				// Add the event to the log
				eventLog.AddEvent(volume.AWSVolumeID, event)

				// Determine if resize is needed
				if IsThresholdExceeded(&volumeState, float64(volume.ResizeThreshold)) {
					DebugPrint(debugMode, "Threshold exceeded for volume, starting resizing process...")

					// Calculate the new size
					currentSize, err := aws.GetAWSDeviceSizeGB(volume)
					if err != nil {
						DebugPrint(debugMode, fmt.Sprintf("Failed to get current size for volume %s: %v\n", volume.AWSVolumeID, err))
						DebugPrint(debugMode, fmt.Sprintf("error: %v", err))
						errorLog[volume.AWSVolumeID]++ // increase error count
					} else {
						var newSize int64
						errorLog[volume.AWSVolumeID] = 0 // Reset the error count on successful operation

						// Check if IncreaseSizeGB is declared in config.yaml
						// will be < 0 if not declaed in config.yaml
						if volume.IncrementSizeGB > 0 {
							newSize = currentSize + int64(volume.IncrementSizeGB)
							DebugPrint(debugMode, fmt.Sprintf("Manually calculated new size for volume %s is %d\n", volume.AWSVolumeID, newSize))
						} else {
							// calculate new size based on percentage as increaseByGB was not specified
							newSize = resize.CalculateNewSize(volume, currentSize)
							DebugPrint(debugMode, fmt.Sprintf("Calculated new size for volume %s is %d\n", volume.AWSVolumeID, newSize))
						}

						DebugPrint(debugMode, "Performing resize...")

						// Perform the resize
						// NOTE: event log logging for resize actions is handled by resize.PerformResize function
						err = resize.PerformResize(volume, newSize, &eventLog)
						if err != nil {
							l.Log(logger.LogError, fmt.Sprintf("Failed to resize volume"), nil)
							DebugPrint(debugMode, fmt.Sprintf(" %s: %v\n", volume.AWSVolumeID, err))
							DebugPrint(debugMode, fmt.Sprintf("error: %v", err))
							errorLog[volume.AWSVolumeID]++ // increase error count
						} else {
							DebugPrint(debugMode, fmt.Sprintf("Successfully resized device: %s from %vGB to %vGB.", volume.AWSDeviceName, currentSize, newSize))
							l.Log(logger.LogInfo, fmt.Sprintf("Successfully resized device: %s from %vGB to %vGB.", volume.AWSDeviceName, currentSize, newSize), nil)
							errorLog[volume.AWSVolumeID] = 0 // Reset the error count on successful operation
						}
					}

				}

			}
			index++
		}

		// Check if there are volumes left to monitor after the for loop
		if len(appRuntime.Configuration.Volumes) == 0 {
			l.Log(logger.LogError, "No more volumes to monitor", nil)
			os.Exit(1)
		}

		// Prunes any events from the eventLog that are >24 hours old.
		PruneAndSleep(&eventLog, appRuntime.Configuration.CheckIntervalSeconds)
	}
}

// main : The entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		l.Log(logger.LogError, "Failed to execute root command", map[string]interface{}{
			"error": err,
		})
		os.Exit(1)
	}
}

// PrintStructFields : Debugging function to print all fields of a struct to terminal
// data : interface{} The struct to print
func PrintStructFields(data interface{}, indent string) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	fmt.Printf("%s%s {\n", indent, t.Name())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.PkgPath == "" { // field is exported
			if value.Kind() == reflect.Struct {
				PrintStructFields(value.Interface(), indent+"   ")
			} else {
				fmt.Printf("%s   %s: %v\n", indent, field.Name, value.Interface())
			}
		} else {
			fmt.Printf("%s   %s: [unexported]\n", indent, field.Name)
		}
	}
	fmt.Printf("%s}\n", indent)
}

// DumpRuntime : Function to print all fields of config.yaml, eventLog, and errorLog
// config : *runtime.Config The config to print
// eventLog : runtime.EventLog The event log to print
// errorLog : make(map[string]int) The error log for each volume
func DumpRuntime(config *runtime.Config, eventLog runtime.EventLog, errorLog map[string]int) {
	DebugPrint(debugMode, "=== CONFIG.YAML ===")
	DebugPrint(debugMode, fmt.Sprintf("Config: %v\n", config))

	DebugPrint(debugMode, "=== EVENT LOG ===")
	for volumeID, events := range eventLog {
		DebugPrint(debugMode, fmt.Sprintf("VolumeID: %s", volumeID))
		DebugPrint(debugMode, fmt.Sprintf("Error Count: %d", errorLog[volumeID]))
		for _, event := range events {
			DebugPrint(debugMode, "Event Details:")
			DebugPrint(debugMode, fmt.Sprintf("%v", event))
		}
	}
}

// InitialiseApp : Initializes the application by creating runtime and configuration.
// Returns: (*runtime.Runtime, *runtime.Config)
func InitialiseApp() (*runtime.Runtime, *runtime.Config) {
	return runtime.InitialiseRuntime(), runtime.InitialiseConfig()
}

// LoadConfig : Function to load configuration values from a file.
// configFile : string The path to the configuration file.
// Returns a slices, an int, and an error. The first slice contains the volumes, the second contains the check intervals in seconds.
func LoadConfig(configFile string) (volumes []runtime.EBSVolumeConfig, checkIntervalSeconds int, err error) {
	volumes, checkIntervalSeconds, err = configutil.GetConfigFromFile(configFile)
	if err != nil {
		l.Log(logger.LogError, "Failed to get config from file", map[string]interface{}{
			"error":      err,
			"configFile": configFile,
		})
		os.Exit(1)
	}
	return volumes, checkIntervalSeconds, err
}

// IsThresholdExceeded : Checks if the disk utilisation of volume state is above the resizeThreshold and prints a message.
// volumeState : *runtime.EBSVolumeState The state of the volume.
// resizeThreshold : float64 The threshold to resize.
// Returns a boolean value indicating if the threshold has been exceeded.
func IsThresholdExceeded(volumeState *runtime.EBSVolumeState, resizeThreshold float64) bool {
	resizeThresholdGB := volumeState.LocalDiskSizeGB * (resizeThreshold / 100.0)

	fmt.Printf("Volume ID: %v\nResize Threshold (%%): %v\nResize Threshold (GB): %v\n", volumeState.AWSVolumeID, resizeThreshold, resizeThresholdGB)

	if volumeState.UsedSpaceGB > resizeThresholdGB {
		// Calculate exceeded value
		exceededBy := volumeState.UsedSpaceGB - resizeThresholdGB
		DebugPrint(debugMode, fmt.Sprintf("Exceeded threshold by %.2f GB\n", exceededBy))
		return true
	} else {
		DebugPrint(debugMode, "Below threshold")
		return false
	}
}

// MonitorVolume : Monitors the volume and checks the state of it.
// monitoredVolume : runtime.EBSVolumeConfig The volume to monitor.
// eventLog : *runtime.EventLog The log of events.
// errorLog : map[string]int The count of errors.
// Returns: (*monitor.EBSVolumeState, error)
func MonitorVolume(monitoredVolume runtime.EBSVolumeConfig, eventLog *runtime.EventLog, errorLog map[string]int) (runtime.EBSVolumeState, error) {
	volumeState, err := monitor.GetVolumeState(monitoredVolume, eventLog)
	if err != nil {
		errorLog[monitoredVolume.AWSVolumeID]++
		return volumeState, err
	}
	return volumeState, err
}

// PruneAndSleep : Prunes stale events from the log and sleeps for check interval.
// eventLog : *runtime.EventLog The log of events.
// checkIntervalSeconds : int The check interval in seconds.
func PruneAndSleep(eventLog *runtime.EventLog, checkIntervalSeconds int) {
	eventLog.PruneStaleEvents()
	time.Sleep(time.Duration(checkIntervalSeconds) * time.Second)
}

// DebugPrint : used to provide conditional printing of debug messages
// Helps with code insight when run with --debug flag
// debugMode : bool - indicates whether to print or not
// message : string - what to print if true
func DebugPrint(debugMode bool, message string) {
	if debugMode {
		l.Log(logger.LogDebug, fmt.Sprintf("DEBUG: "+message), nil)
	}
}
