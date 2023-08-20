package resize

import (
	"ebs-monitor/aws"
	"ebs-monitor/filesystem"
	"ebs-monitor/runtime"
	"fmt"
	"time"
)

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
func PerformResize(volume runtime.EBSVolumeConfig, newSize int64, log *runtime.EventLog) error {
	// Get the local mount point of the EBS volume
	localMountPoint, err := filesystem.GetLocalMountPoint(volume.AWSVolumeID)
	if err != nil {
		return fmt.Errorf("failed to get local mount point of volume '%v'. error: %w", volume.AWSDeviceName, err)
	}

	// Get the current size of the AWS EBS volume
	currentAWSVolumeSize, err := aws.GetAWSDeviceSizeGB(volume)
	if err != nil {
		return fmt.Errorf("failed to get the size of the EBS volume '%v' in AWS. error: %w", volume.AWSDeviceName, err)
	}

	// Get the current size of the local filesystem
	currentLocalDiskSize, err := filesystem.GetLocalDiskSizeGB(localMountPoint)
	if err != nil {
		return fmt.Errorf("failed to get the size of the local filesystem for '%v'. error: %w", localMountPoint, err)
	}

	var awsResizeErr error

	/*
		######################################
			Resize AWS volume
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
	awsResizeErr = aws.ResizeVolume(volume, newSize)
	if awsResizeErr == nil {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateVolumeResizeActionEvent(volumeAction, true))
	} else {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateVolumeResizeActionEvent(volumeAction, false))
	}

	// Adding sleep to fix issue attempting filesystem resize immediately after EBS resize action.
	time.Sleep(time.Second * 30)

	/*
		######################################
			Resize local filesystem volume
		######################################
	*/
	// Initialize FilesystemResize struct
	fsAction := runtime.FilesystemResize{
		StartTime:       time.Now(),
		AWSVolumeID:     volume.AWSVolumeID,
		AWSDeviceName:   volume.AWSDeviceName,
		LocalMountPoint: localMountPoint,
		AWSVolumeSize:   float64(currentAWSVolumeSize),
		OriginalSizeGB:  currentLocalDiskSize,
		NewSize:         float64(newSize),
	}

	// Resize the file system on the EBS volume
	fsResizeErr := filesystem.ResizeFilesystem(volume)
	if fsResizeErr == nil {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, true))
	} else {
		(*log)[volume.AWSVolumeID] = append((*log)[volume.AWSVolumeID], runtime.CreateFSActionEvent(fsAction, false))
	}

	// Check if there was an error in either resize operation
	if awsResizeErr != nil || fsResizeErr != nil {
		var errMsg string

		if awsResizeErr != nil {
			errMsg += fmt.Sprintf("AWS resize error: %v. ", awsResizeErr)
		}
		if fsResizeErr != nil {
			errMsg += fmt.Sprintf("Filesystem resize error: %v. ", fsResizeErr)
		}

		return fmt.Errorf(errMsg)
	}

	return nil
}
