package resize

import (
	"ebs-monitor/aws"
	"ebs-monitor/filesystem"
	"ebs-monitor/logger"
	"ebs-monitor/runtime"
	"fmt"
	"time"
)

// Initialise logger
var l = logger.NewLogger()

// CalculateNewSize : Calculates the new size of the volume based on the given configuration
// config : runtime.EBSVolumeConfig : Configuration of the EBS volume
// currentSize : int64 : The current size of the volume in GiB
// returns : int64 : The new size of the volume in GiB
func CalculateNewSize(config runtime.EBSVolumeConfig, currentSize int64) int64 {
	// Calculate the increment size in GiB
	incrementSize := currentSize * int64(config.IncrementSizePercent) / 100

	// Calculate the new size
	newSize := currentSize + incrementSize

	return newSize
}

// PerformResize : Performs the resize operation on the volume after checking
// the EBS volume size and comparing it with the filesystem size
// config : runtime.EBSVolumeConfig : Configuration of the EBS volume
// newSize : int64 : The new size of the volume in GiB
// returns : error : Any error that occurred during operation, nil if operation was successful
func PerformResize(volume runtime.EBSVolumeConfig, newSize int64, log *runtime.EventLog) (bool, bool, error) {

	// Tracks the success of resize actions taken
	awsResized := false
	fsResized := false

	// Get the local mount point of the EBS volume
	localMountPoint, err := filesystem.GetLocalMountPoint(volume.AWSVolumeID)
	if err != nil {
		return awsResized, fsResized, fmt.Errorf("failed to get local mount point of volume '%v'. error: %w", volume.AWSDeviceName, err)
	}
	fmt.Printf("Successfully fetched local mount point: %v\n", localMountPoint)

	fmt.Println("STEP 1 - Attempting Filesystem Extension...")
	// STEP 1 - Attempt Filesystem Extension First
	// If successful return nil, otherwise proceed with EBS volume resize action
	// Initialize FilesystemResize struct for logging history
	fsAction := runtime.FilesystemResize{
		StartTime:       time.Now(),
		AWSVolumeID:     volume.AWSVolumeID,
		AWSDeviceName:   volume.AWSDeviceName,
		LocalMountPoint: localMountPoint,
		NewSize:         float64(newSize),
	}

	// Attempt extending filesystem
	fsResizeErr := filesystem.ResizeFilesystem(volume)

	// Add attempt to history
	if fsResizeErr == nil {
		fmt.Println("Filesystem resize was successful, increased size to: ", newSize)
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, true))
	} else {
		fmt.Println("Failed to resize the filesystem on the first attempt. Error: ", fsResizeErr.Error())
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, false))
	}

	// Get the current size of the AWS EBS volume
	currentAWSVolumeSize, err := aws.GetAWSDeviceSizeGB(volume)
	if err != nil {
		return awsResized, fsResized, fmt.Errorf("failed to get the size of the EBS volume '%v' in AWS. error: %w", volume.AWSDeviceName, err)
	}

	// Get the current size of the local filesystem
	currentLocalDiskSize, err := filesystem.GetLocalDiskSizeGB(localMountPoint)
	if err != nil {
		return awsResized, fsResized, fmt.Errorf("failed to get the size of the local filesystem for '%v'. error: %w", localMountPoint, err)
	}

	// If successful return nil
	if fsResizeErr == nil {
		fmt.Println("Filesystem resize was successful. Exiting early from PerformResize.")
		// Log success and return details of volume
		// Dropped level to Debug to prevent duplicate SNS notifications
		l.Log(logger.LogDebug, "Filesystem resized successfully.", map[string]interface{}{
			"AWS Volume ID":     volume.AWSVolumeID,
			"AWS Device Name":   volume.AWSDeviceName,
			"Local Mount Point": localMountPoint,
			"AWS Region":        volume.AWSRegion,
			"EBS Volume Size":   currentAWSVolumeSize,
			"Local Disk Size":   currentLocalDiskSize,
		})
		fsResized = true
	}

	fmt.Println("STEP 2 - Checking AWS Volume State...")
	// STEP 2 -  Check AWS Volume State - can we extend it?
	// is the volume in an optimizing state? if yes, return error
	isOptimizing, err := aws.CheckVolumeState(volume)
	fmt.Println("Optimizing state return: ", isOptimizing)
	if err != nil {
		fmt.Println("Failed to check if volume is optimizing.")
		return awsResized, fsResized, err
	}
	if isOptimizing {
		fmt.Println("Volume is optimizing, aborting")
		return awsResized, fsResized, fmt.Errorf("volume %v:%v is in optimizing state. Unable to attempt resize action", volume.AWSVolumeID, volume.AWSDeviceName)
	}

	fmt.Println("STEP 3: Resizing AWS volume...")

	/*
		######################################
			STEP 3: Resize AWS volume
		######################################
	*/

	// Initialize EBSVolumeResize struct
	volumeAction := runtime.EBSVolumeResize{
		StartTime:      time.Now(),
		AWSVolumeID:    volume.AWSVolumeID,
		AWSDeviceName:  volume.AWSDeviceName,
		AWSRegion:      volume.AWSRegion,
		OriginalSizeGB: float64(currentAWSVolumeSize),
		NewSize:        float64(newSize),
	}

	// Resize the EBS volume in AWS
	// Return error if action fails
	awsResizeErr := aws.ResizeVolume(volume, newSize)
	if awsResizeErr == nil {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateVolumeResizeActionEvent(volumeAction, true))
		awsResized = true
	} else {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateVolumeResizeActionEvent(volumeAction, false))
		return awsResized, fsResized, awsResizeErr
	}

	// Adding sleep to fix issue attempting filesystem resize immediately after EBS resize action.
	fmt.Println("Adding sleep (60s) before attempting filesystem resize...")
	time.Sleep(time.Second * 60)

	fmt.Println("STEP 4: Resizing local filesystem volume...")

	/*
		##############################################
			STEP 4: Resize local filesystem volume
		##############################################
	*/
	// Initialize FilesystemResize struct
	fsAction = runtime.FilesystemResize{
		StartTime:       time.Now(),
		AWSVolumeID:     volume.AWSVolumeID,
		AWSDeviceName:   volume.AWSDeviceName,
		LocalMountPoint: localMountPoint,
		AWSVolumeSize:   float64(currentAWSVolumeSize),
		OriginalSizeGB:  currentLocalDiskSize,
		NewSize:         float64(newSize),
	}

	// Resize the file system on the EBS volume
	// Return error if action fails
	fsResizeErr = filesystem.ResizeFilesystem(volume)
	if fsResizeErr == nil {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, true))
		fsResized = true
	} else {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, false))
		return awsResized, fsResized, fsResizeErr
	}

	fmt.Println("PerformResize function completed.")
	return awsResized, fsResized, nil
}
