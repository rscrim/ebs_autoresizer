package aws

import (
	"context"
	"ebs-monitor/runtime"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// getCurrentRegion fetches the current region from EC2 instance metadata using the AWS SDK for Go V2.
// returns : string : AWS region where the instance is located
// returns : error : return an error if any occur during the process
func getCurrentRegion() (string, error) {
	// Load the default SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	// Create a new EC2 Instance Metadata Service client
	client := imds.NewFromConfig(cfg)

	// Use the client to retrieve the region of the instance
	response, err := client.GetRegion(context.TODO(), &imds.GetRegionInput{})
	if err != nil {
		log.Printf("Unable to retrieve the region from the EC2 instance: %v\n", err)
		return "", err
	}

	return response.Region, nil
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

// getInstanceID : Fetches the instance ID of the current instance using AWS SDK's IMDS client
// Returns: string : The instance ID of the current instance
// error : error : An error that occurred while getting the instance ID, or nil if no error occurred
func getInstanceID() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	client := imds.NewFromConfig(cfg)
	resp, err := client.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return "", err
	}

	return resp.InstanceID, nil
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

// ChatbotMessage is a struct that reflects the message format for Chatbot to post to Slack
type ChatbotMessage struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	NextSteps   []string `json:"nextSteps,omitempty"`
}

// PublishToSNS publishes a structured message to an SNS topic.
// arn: string - ARN of the SNS topic.
// snsRegion: string - AWS region of the SNS topic.
// message: ChatbotMessage - The structured message to be published.
// returns: error - Returns an error if any occur during the process.
func PublishToSNS(arn string, snsRegion string, messageDescription string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(snsRegion))
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %v", err)
	}

	// Get AWS account number
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("unable to get AWS account number, %v", err)
	}
	accountNumber := awsv2.ToString(identity.Account)

	// Get instance hostname
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("unable to get hostname, %v", err)
	}

	// Get region of EC2 instance running ebs-monitor.service
	instanceRegion, err := getCurrentRegion()

	if err != nil {
		return fmt.Errorf("unable to get instance region, %v", err)
	}

	// Fetch the versions of ebs-monitor.service
	runningVersion, latestVersion, err := GetEBSVersions()
	if err != nil {
		// Handle the error or set versions to a default or error value
		runningVersion, latestVersion = "unknown", "unknown"
		fmt.Println("Error: ", err)
	}

	// Construct enriched message
	msgContent := ChatbotMessage{
		Title:       fmt.Sprintf(":no_entry: ebsmon-alert: %s", hostname),
		Description: messageDescription,
		NextSteps: []string{
			fmt.Sprintf("Hostname: %s", hostname),
			fmt.Sprintf("Account Number: %s", accountNumber),
			fmt.Sprintf("Region: %s", instanceRegion),
			fmt.Sprintf("Running Version: %s", runningVersion),
			fmt.Sprintf("Latest Available Version: %s", latestVersion),
		},
	}

	// Check if an update is needed and include a warning message if so
	if runningVersion < latestVersion {
		msgContent.NextSteps = append(msgContent.NextSteps, fmt.Sprintf(":warning: ebs-monitor needs to be updated from version %s to %s", runningVersion, latestVersion))
	}
	if runningVersion > latestVersion {
		msgContent.NextSteps = append(msgContent.NextSteps, fmt.Sprintf(":grey_exclamation: ebs-monitor is running a pre-release version... this may lead to issues.\n\t\tRunning: %s\n\t\tAvailable: %s", runningVersion, latestVersion))
	}

	// Create message struct to post
	message := map[string]interface{}{
		"version": "1.0",
		"source":  "custom",
		"content": msgContent,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("unable to marshal message to JSON, %v", err)
	}

	// Publish the enriched message to SNS
	client := sns.NewFromConfig(cfg)
	_, err = client.Publish(context.TODO(), &sns.PublishInput{
		Message:  aws.String(string(messageJSON)),
		TopicArn: aws.String(arn),
	})
	if err != nil {
		return fmt.Errorf("unable to publish message to SNS, %v", err)
	}

	return nil
}

// CheckVolumeState checks the modification state of the specified EBS volume.
// It returns true if the volume is in the 'optimizing' state, false otherwise.
// config : runtime.EBSVolumeConfig : configuration of the EBS volume
// returns : bool : returns true if the volume is in the 'optimizing' state, false otherwise
// returns : error : returns an error if any occur during the process
func CheckVolumeState(config runtime.EBSVolumeConfig) (bool, error) {
	// Create a new session
	svc := NewSession(config.AWSRegion)

	// Define input for DescribeVolumesModifications call
	input := &ec2.DescribeVolumesModificationsInput{
		VolumeIds: []*string{
			aws.String(config.AWSVolumeID),
		},
	}

	// Call DescribeVolumesModifications API
	result, err := svc.DescribeVolumesModifications(input)
	if err != nil {
		// Check for the specific error of no modifications
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVolumeModification.NotFound":
				return false, nil // No modifications, return false with no error
			default:
				return false, fmt.Errorf("failed to get volume modification information from AWS. error: %w", err)
			}
		} else {
			return false, fmt.Errorf("failed to get volume modification information from AWS. error: %w", err)
		}
	}

	// Check if volume modification was found
	if len(result.VolumesModifications) == 0 {
		return false, fmt.Errorf("failed to find volume modification information. error: %w", err)
	}

	// Check the modification state of the volume
	if *result.VolumesModifications[0].ModificationState == ec2.VolumeModificationStateOptimizing {
		return true, nil
	}

	return false, nil
}

// -----------------------------------------------------------------
// IT IS NOT A GOOD PLACE TO PUT THIS FUNCTIONHERE
// BUT I COULDN'T THINK OF WHERE ELSE FOR IT TO GO WITHOUT INTRODUCING
// CIRCULAR DEPENDENCIES.. SO HERE WE ARE
// -----------------------------------------------------------------

// GetEBSVersions : fetches the running version and the latest available version of ebs-monitor.service.
// returns : string : Running version of the ebs-monitor.service
// returns : string : Latest available version for installation
// returns : error : Potential errors during the operation
func GetEBSVersions() (string, string, error) {
	// Get the running version
	cmd := exec.Command("ebsmon", "--version")
	runningVersionBytes, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	runningVersion := strings.TrimSpace(string(runningVersionBytes))

	// Get the version details using apt-cache policy
	cmd = exec.Command("apt-cache", "policy", "ebs-monitor")
	aptOutputBytes, err := cmd.Output()
	if err != nil {
		return runningVersion, "", err
	}
	aptOutput := string(aptOutputBytes)

	// Extract the installed version
	reInstalled := regexp.MustCompile(`Installed: (\d+\.\d+\.\d+)`)
	matchesInstalled := reInstalled.FindStringSubmatch(aptOutput)
	if len(matchesInstalled) < 2 {
		return runningVersion, "", fmt.Errorf("could not extract installed version from apt output")
	}
	installedVersion := matchesInstalled[1]

	// Extract the candidate version
	reCandidate := regexp.MustCompile(`Candidate: (\d+\.\d+\.\d+)`)
	matchesCandidate := reCandidate.FindStringSubmatch(aptOutput)
	if len(matchesCandidate) < 2 {
		return installedVersion, "", fmt.Errorf("could not extract candidate version from apt output")
	}
	candidateVersion := matchesCandidate[1]

	return installedVersion, candidateVersion, nil
}
