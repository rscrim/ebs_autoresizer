package configutil

import (
	"ebs-monitor/aws"
	"ebs-monitor/runtime"
	"errors"
	"fmt"

	"github.com/spf13/viper"
)


// GetConfigFromFile : reads a configuration file, parses its content, and returns runtime components.
// Includes configuration validation for each volume and lookups for missing, important data.
// Volume will not be included if Vol-ID and Device name are missing.
// filename : string name of the file to read
// returns : []runtime.EBSVolumeConfig volume configurations
// returns : time.Duration check interval
// returns : error potential errors
func GetConfigFromFile(filename string) ([]runtime.EBSVolumeConfig, int, error) {
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		return nil, 0, fmt.Errorf("failed to read the configuration file: %v. error: %w", filename, err)
	}
	var cfg runtime.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal the configuration. error: %w", err)
	}
	if err := ValidateConfig(&cfg); err != nil {
		return nil, 0, fmt.Errorf("failed to validate the application configuration. error: %w", err)
	}
	validVolumes := make([]runtime.EBSVolumeConfig, 0)
	for _, volume := range cfg.Volumes {
		if checkMinimumFields(volume) {
			validVolumes = append(validVolumes, volume)
		}
	}

	return validVolumes, cfg.CheckIntervalSeconds, nil
}

// checkMinimumFields : checks if a volume configuration is valid
// volume : runtime.EBSVolumeConfig : volume configuration to validate
// returns : bool : validity of the volume configuration
func checkMinimumFields(volume runtime.EBSVolumeConfig) bool {
	if (volume.AWSVolumeID == "" && volume.AWSDeviceName == "") ||
		(volume.IncrementSizeGB == 0 && volume.IncrementSizePercent == 0) ||
		volume.ResizeThreshold == 0 {
		return false
	}
	return true
}

// validateConfig : validates the configuration and adds missing information
// from the config.yaml.
// config : Config : configuration to validate
// returns : error : potential errors
func ValidateConfig(config *runtime.Config) error {
	for i := range config.Volumes {
		if err := validateVolume(&config.Volumes[i]); err != nil {
			return err
		}
	}
	return nil
}

/*
-------------------------
Helper functions to validate the config
-------------------------
*/

// validateAWSVolumeID : checks if a string matches AWS's volume ID format.
// id : string volume ID to validate
// region : string : AWS region where the volume is located
// returns : error potential errors
func validateAWSVolumeID(id, region string) error {
	valid, err := aws.ValidateVolumeID(id, region)
	if err != nil {
		return fmt.Errorf("failed to validate aws volume id. error: %w", err)
	}
	if !valid {
		return errors.New("invalid AWS volume ID")
	}
	return nil
}

// validateAWSDeviceName : checks if a string matches AWS's device name format.
// name : string device name to validate
// region : string : AWS region where the device is located
// returns : error potential errors
func validateAWSDeviceName(name, region string) error {
	valid, err := aws.ValidateDeviceName(name, region)
	if err != nil {
		return fmt.Errorf("failed to validate aws device name. error: %w", err)
	}
	if !valid {
		return errors.New("invalid AWS device name")
	}
	return nil
}

// validateAWSRegion : checks if a string matches AWS's region format.
// region : string : region to validate
// returns : error : returns an error if the region is invalid
func validateAWSRegion(region string) error {
	valid, err := aws.ValidateRegion(region)
	if err != nil {
		return fmt.Errorf("failed to validate aws region. error: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid AWS region: %s", region)
	}
	return nil
}

// validatePositiveInt : checks if an int is greater than or equal to 0.
// num : int number to validate
// returns : error potential errors
func validatePositiveInt(num int) error {
	if num < 0 {
		return errors.New("value should be greater than or equal to 0")
	}
	return nil
}

// validateVolume : validates the volume configuration
// volume : runtime.EBSVolumeConfig : volume configuration to validate
// returns : error : potential errors
func validateVolume(volume *runtime.EBSVolumeConfig) error {
	// Try to validate the region from the config
	err := validateAWSRegion(volume.AWSRegion)
	if err != nil {
		// If the region is invalid, lookup the region from the EC2 instance metadata
		volume.AWSRegion, err = aws.GetLocalRegion() // assuming aws.GetLocalRegion() returns the local region
		if err != nil {
			return fmt.Errorf("failed to get local region. error: %w", err)
		}
	}

	// Use the region (either from the config or the local region) for the rest of the validations
	// If AWSVolumeID is provided and device name is omitted, perform lookup
	if volume.AWSVolumeID != "" {
		if err := validateAWSVolumeID(volume.AWSVolumeID, volume.AWSRegion); err != nil {
			return err
		}

		if volume.AWSDeviceName == "" {
			deviceName, err := aws.GetDeviceNameByVolumeID(volume.AWSVolumeID, volume.AWSRegion)
			if err != nil {
				return fmt.Errorf("failed to get device name for volume ID: %v, error: %w", volume.AWSVolumeID, err)
			}

			volume.AWSDeviceName = deviceName
		}

		// if AWSVolumeID is omitted but device name is provided, perform
	} else if volume.AWSDeviceName != "" {
		if err := validateAWSDeviceName(volume.AWSDeviceName, volume.AWSRegion); err != nil {
			return err
		}

		volumeID, err := aws.GetVolumeIDByDeviceName(volume.AWSDeviceName, volume.AWSRegion)
		if err != nil {
			return fmt.Errorf("failed to get volume ID for device name: %v, error: %w", volume.AWSDeviceName, err)
		}

		volume.AWSVolumeID = volumeID
	}

	if err := validatePositiveInt(volume.IncrementSizeGB); err != nil {
		return err
	}
	if err := validatePositiveInt(volume.IncrementSizePercent); err != nil {
		return err
	}
	if err := validatePositiveInt(volume.ResizeThreshold); err != nil {
		return err
	}
	return nil
}
