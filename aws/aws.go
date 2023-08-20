package aws

import (
	"ebs-monitor/runtime"
	"fmt"
	"io"
	"net/http"

	// replace this with the actual import path
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// NewSession : creates a new EC2 service client
// region : string : AWS region for the client
// returns : *ec2.EC2 : returns an EC2 service client
func NewSession(region string) *ec2.EC2 {
	// Create a new session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	// Create an EC2 service client
	return ec2.New(sess)
}

// GetVolume : retrieves an EBS volume using the provided runtime.EBSVolumeConfig
// config : runtime.EBSVolumeConfig : configuration of the EBS volume
// returns : *ec2.Volume : returns the EBS volume
// returns : error : returns an error if any occur during the process
func GetVolume(config runtime.EBSVolumeConfig) (*ec2.Volume, error) {
	// Create a new session
	svc := NewSession(config.AWSRegion)

	// Define input for DescribeVolumes call
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(config.AWSVolumeID),
		},
	}

	// Call DescribeVolumes API
	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume information from aws. error: %w", err)
	}

	// Check if volume was found
	if len(result.Volumes) == 0 {
		return nil, fmt.Errorf("failed to find volume information. error: %w", err)
	}

	// Return the found volume
	return result.Volumes[0], nil
}

// GetAWSDeviceSizeGB : retrieves the size of the EBS volume specified in the runtime.EBSVolumeConfig in GiB
// config : runtime.EBSVolumeConfig : configuration of the EBS volume
// returns : int64 : returns the size of the volume in GiB
// returns : error : returns an error if any occur during the process
func GetAWSDeviceSizeGB(config runtime.EBSVolumeConfig) (int64, error) {
	// Retrieve the volume
	volume, err := GetVolume(config)
	if err != nil {
		return 0, fmt.Errorf("failed to get volume information. error: %w", err)
	}

	// Return the size of the volume
	return *volume.Size, nil
}

// GetVolumeState : retrieves the state of the EBS volume specified in the runtime.EBSVolumeConfig
// config : runtime.EBSVolumeConfig : configuration of the EBS volume
// returns : string : returns the state of the volume
// returns : error : returns an error if any occur during the process
func GetVolumeState(config runtime.EBSVolumeConfig) (string, error) {
	// Retrieve the volume
	volume, err := GetVolume(config)
	if err != nil {
		return "", fmt.Errorf("failed to get volume state. error: %w", err)
	}

	// Return the state of the volume
	return *volume.State, nil
}

// GetAllRegions : retrieves all AWS regions
// returns : []string : slice of all AWS region names
// returns : error : returns an error if any occur during the process
func GetAllRegions() ([]string, error) {
	// Create a session
	sess := NewSession("us-east-1")

	// Call EC2 DescribeRegions API
	resultRegions, err := sess.DescribeRegions(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve aws regions. error: %v", err)
	}

	// Collect all region names
	var regions []string
	for _, region := range resultRegions.Regions {
		regions = append(regions, *region.RegionName)
	}

	return regions, nil
}

// ValidateVolumeID : checks if the provided Volume ID is valid
// volumeID : string : AWS EBS volume ID to validate
// region : string : AWS region where the volume is located
// returns : bool : returns true if the Volume ID is valid, false otherwise
// returns : error : returns an error if any occur during the process
func ValidateVolumeID(volumeID, region string) (bool, error) {
	// Create a new session
	svc := NewSession(region)

	// Define input for DescribeVolumes call
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}

	// Call DescribeVolumes API
	_, err := svc.DescribeVolumes(input)
	if err != nil {
		return false, fmt.Errorf("failed to call DescribeVolumes API to validate volume ID. error: %w", err)
	}

	return true, nil
}

// getInstanceID : Fetches the instance ID of the current instance from AWS EC2 metadata
// Returns: string : The instance ID of the current instance
// error : error : An error that occurred while getting the instance ID, or nil if no error occurred
func getInstanceID() (string, error) {
	resp, err := http.Get("http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// GetVolumeIDByDeviceName : Fetches the volume ID attached to a specific device name of the current instance
// deviceName : string : Device name attached to the volume
// region : string : AWS region name
// Returns: string : The volume ID attached to the device name in the current instance
// error : error : An error that occurred while getting the volume ID, or nil if no error occurred
func GetVolumeIDByDeviceName(deviceName, region string) (string, error) {
	// Get the instance ID from metadata service
	instanceID, err := getInstanceID()
	if err != nil {
		return "", fmt.Errorf("failed to get instance ID: %w", err)
	}

	// Create a new session
	svc := NewSession(region)

	// Create input configuration
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	// Call DescribeInstances API
	resp, err := svc.DescribeInstances(input)
	if err != nil {
		return "", fmt.Errorf("failed to get instance information from AWS: %w", err)
	}

	// Loop over reservations and instances
	for _, res := range resp.Reservations {
		for _, inst := range res.Instances {
			// Loop over instance block device mappings
			for _, bd := range inst.BlockDeviceMappings {
				if *bd.DeviceName == deviceName {
					return *bd.Ebs.VolumeId, nil
				}
			}
		}
	}

	// Return error if no volume found
	return "", fmt.Errorf("no volume found with device name %v", deviceName)
}

// GetDeviceNameByVolumeID : retrieves the device name of the EBS volume attached to an EC2 instance
// volumeID : string : AWS EBS volume ID
// region : string : AWS region where the volume is located
// returns : string : returns the device name
// returns : error : returns an error if any occur during the process
func GetDeviceNameByVolumeID(volumeID, region string) (string, error) {
	// Create a new session
	svc := NewSession(region)

	// Call DescribeInstances API
	resp, err := svc.DescribeInstances(nil)
	if err != nil {
		return "", fmt.Errorf("failed to get instance information from AWS. error: %w", err)
	}

	// Loop over reservations and instances
	for _, res := range resp.Reservations {
		for _, inst := range res.Instances {
			// Loop over instance block device mappings
			for _, bd := range inst.BlockDeviceMappings {
				if *bd.Ebs.VolumeId == volumeID {
					return *bd.DeviceName, nil
				}
			}
		}
	}

	// Return error if no device name found
	return "", fmt.Errorf("no device name found for volume ID %v", volumeID)
}

// ValidateDeviceName : checks if the provided Device Name is valid
// deviceName : string : AWS Device Name to validate
// region : string : AWS region where the device is located
// returns : bool : returns true if the Device Name is valid, false otherwise
// returns : error : returns an error if any occur during the process
func ValidateDeviceName(deviceName, region string) (bool, error) {
	// Create a new session
	svc := NewSession(region)

	// Define input for DescribeVolumes call
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("block-device-mapping.device-name"),
				Values: []*string{
					aws.String(deviceName),
				},
			},
		},
	}

	// Call DescribeInstances API
	_, err := svc.DescribeInstances(input)
	if err != nil {
		return false, fmt.Errorf("failed to get getting instance information from AWS. error: %w", err)
	}

	return true, nil
}

// ValidateRegion : checks if the provided Region is valid
// region : string : AWS Region to validate
// returns : bool : returns true if the Region is valid, false otherwise
// returns : error : returns an error if any occur during the process
func ValidateRegion(region string) (bool, error) {
	// Get all regions
	regions, err := GetAllRegions()
	if err != nil {
		return false, err
	}

	// Check if the provided region is in the list of regions
	for _, r := range regions {
		if r == region {
			return true, nil
		}
	}

	// If the provided region is not found in the list of regions, return false
	return false, nil
}

// GetLocalRegion : retrieves the region of the local EC2 instance from its metadata
// returns : region : string : the region of the local EC2 instance
// returns : err : error : any error that occurs during the process
func GetLocalRegion() (string, error) {
	// Create a new session
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	// Create a new EC2Metadata client
	ec2metadataSvc := ec2metadata.New(sess)

	// Retrieve the region of the local EC2 instance
	region, err := ec2metadataSvc.Region()
	if err != nil {
		return "", err
	}

	return region, nil
}

// ResizeVolume: Resizes an EBS volume.
// config: runtime.EBSVolumeConfig - Configuration for the EBS volume.
// newSize: int64 - New size for the EBS volume.
// error: error - Returns an error if there was a problem resizing the volume or if the timeout is reached while waiting for the volume to resize.
func ResizeVolume(config runtime.EBSVolumeConfig, newSize int64) error {
	// Create a session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion)},
	)

	if err != nil {
		return fmt.Errorf("failed to get region information from AWS. error: %w", err)
	}

	// Create a EC2 service client
	svc := ec2.New(sess)

	// Modifying the EBS volume
	modifyOutput, err := svc.ModifyVolume(&ec2.ModifyVolumeInput{
		VolumeId: aws.String(config.AWSVolumeID),
		Size:     aws.Int64(int64(newSize)),
	})

	if err != nil {
		return fmt.Errorf("failed to modify ebs volume in aws. error: %w", err)
	}

	// Waiting for the volume to enter the 'optimizing' state
	err = svc.WaitUntilVolumeInUse(&ec2.DescribeVolumesInput{
		VolumeIds: []*string{modifyOutput.VolumeModification.VolumeId},
	})

	if err != nil {
		return fmt.Errorf("failed to wait for volume to enter 'in-use' state again. error: %w", err)
	}

	return nil
}
