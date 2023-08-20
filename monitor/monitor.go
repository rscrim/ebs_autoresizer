package monitor

import (
	"ebs-monitor/aws"
	"ebs-monitor/filesystem"
	"ebs-monitor/runtime"
	"fmt"
)

// GetVolumeState : gathers information on a specific volume and performs error handling.
// volumeConfig : runtime.EBSVolumeConfig configuration of the volume to gather state from
// returns : runtime.EBSVolumeState gathered volume state
// returns : error potential errors
func GetVolumeState(volumeConfig runtime.EBSVolumeConfig, eventLog *runtime.EventLog) (runtime.EBSVolumeState, error) {
	state := runtime.InitialiseEBSVolumeState()

	// Get AWS VolumeID & DeviceName
	state.AWSVolumeID = volumeConfig.AWSVolumeID
	state.AWSDeviceName = volumeConfig.AWSDeviceName

	// Get LocalMountPoint
	mnt, err := filesystem.GetLocalMountPoint(state.AWSVolumeID)
	if err != nil {
		return state, fmt.Errorf("failed to get local mount point information for '%v'. error: %w", state.AWSDeviceName, err)
	}
	state.LocalMountPoint = mnt

	// Get AWS Device Size in GB
	devGB, err := aws.GetAWSDeviceSizeGB(volumeConfig)
	if err != nil {
		return state, fmt.Errorf("failed to get device size for '%v'. error: %w", state.AWSDeviceName, err)
	}
	state.AWSDeviceSizeGB = float64(devGB)

	// Get Local Device Size in GB
	mntGB, err := filesystem.GetLocalDiskSizeGB(mnt)
	if err != nil {
		return state, fmt.Errorf("failed to get local disk size for '%v'. error: %w", mnt, err)
	}
	state.LocalDiskSizeGB = mntGB

	// Get used space
	used, err := filesystem.GetUsedSpaceGB(mnt)
	if err != nil {
		return state, fmt.Errorf("failed to get disk utilization for '%v'. error: %w", mnt, err)
	}
	state.UsedSpaceGB = used

	return state, nil
}
