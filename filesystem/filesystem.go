package filesystem

import (
	"ebs-monitor/runtime"
	"fmt"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/disk"
)

// GetLocalMountPoint : Converts the AWS device name to the local device name format.
// volumeID : string : The AWS device name.
// Returns: string : the local device name of the volume, or an error if one occurred.
func GetLocalMountPoint(volumeID string) (string, error) {
	// If volumeID starts with "vol-", trim the prefix
	if strings.HasPrefix(volumeID, "vol-") {
		volumeID = strings.TrimPrefix(volumeID, "vol-")
	}

	// Run the "lsblk -o NAME,MOUNTPOINT,SERIAL" command
	cmd := exec.Command("lsblk", "-o", "NAME,MOUNTPOINT,SERIAL")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute '%v' command on host. error: %w", cmd, err)
	}

	// Split the output into lines
	lines := strings.Split(string(output), "\n")

	// Iterate over the lines
	for _, line := range lines {
		// If this line contains the volume ID
		if strings.Contains(line, volumeID) {
			// Split the line into fields and return the second field (the local mount point)
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[1], nil
			}
		}
	}

	// The volume ID was not found in the output
	return "", fmt.Errorf("volume ID %s not found", volumeID)
}

// GetDeviceFromMountPoint : Fetches the local device name corresponding to a given mount point.
// mountPoint : string : The mount point from which to find the corresponding local device name.
// Returns : string : Local device name.
// Returns : error  : Any error that occurred during operation, nil if operation was successful.
func GetDeviceFromMountPoint(mountPoint string) (string, error) {
	// Get the device associated with the mount point
	cmd := exec.Command("df", "-h", mountPoint, "--output=source")
	output, err := cmd.CombinedOutput()

	// Handle any errors during the command execution
	if err != nil {
		return "", fmt.Errorf("failed to execute'%v' command on host. error: %w", cmd, err)
	}

	// Process the output to get the device name
	device := strings.TrimSpace(string(output))
	lines := strings.Split(device, "\n")
	// Ensure we have at least two lines (the header and the actual value)
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected output from '%v' command, got: %s", cmd, device)
	}
	// The actual device name is on the second line
	device = lines[1]

	return device, nil
}

// GetFileSystemType fetches the file system type of the given mount point.
// mountPoint : string : The mount point whose file system type is required.
// Returns : string : File system type.
// Returns : error : Any error that occurred during operation, nil if operation was successful.
func GetFileSystemType(mountPoint string) (string, error) {
	device, err := GetDeviceFromMountPoint(mountPoint)
	if err != nil {
		return "", err
	}

	// Use 'lsblk' to get the filesystem type of the device
	cmd := exec.Command("lsblk", "-f", device, "-o", "FSTYPE")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute '%v' command on host. error: %w", cmd, err)
	}

	// Process the output to get the filesystem type
	fsType := strings.TrimSpace(string(output))
	lines := strings.Split(fsType, "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected output from '%v' command, got: %s", cmd, fsType)
	}
	// The filesystem type is on the second line
	fsType = lines[1]

	return fsType, nil
}

// ResizeFileSystemByType : Resizes the file system based on its type.
// filesystem : string : The type of the file system.
// mountPoint : string : The mount point whose file system needs to be resized.
// Returns : error : Any error that occurred during operation, nil if operation was successful.
func ResizeFileSystemByType(filesystem, mountPoint string) error {
	device, err := GetDeviceFromMountPoint(mountPoint)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch filesystem {
	case "ext4":
		cmd = exec.Command("resize2fs", device)
	case "xfs":
		cmd = exec.Command("xfs_growfs", device)
	default:
		return fmt.Errorf("unsupported file system type: %s", filesystem)
	}

	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run '%v' filesystem resizing command on host. error: %w", cmd, err)
	}

	return nil

}

// ResizeFilesystem : Resizes the filesystem of a given volume to maximum available space.
// volume : EBSVolumeConfig : Configuration related to EBS volume.
// Returns : error Any error that occurred during resizing, or nil if resizing was successful.
func ResizeFilesystem(volume runtime.EBSVolumeConfig) error {
	// Get local mount point based on AWS device name
	localMountPoint, err := GetLocalMountPoint(volume.AWSDeviceName)
	if err != nil {
		return err // error
	}

	// Get the filesystem type
	filesystem, err := GetFileSystemType(localMountPoint)
	if err != nil {
		return err
	}

	// Resize the filesystem based on its type
	err = ResizeFileSystemByType(filesystem, localMountPoint)
	if err != nil {
		return err
	}

	return nil
}

// GetLocalDiskSizeGB : retrieves the LocalDiskSizeGB.
// returns : float64 LocalDiskSizeGB
// returns : error potential errors
func GetLocalDiskSizeGB(localMountPoint string) (float64, error) {
	usageStat, err := disk.Usage(localMountPoint)
	if err != nil {
		return -1, fmt.Errorf("failed to get disk usage for '%v'. error: %w", localMountPoint, err)
	}

	// Convert disk usage values to GB
	LocalDiskSizeGB := float64(usageStat.Total) / (1024 * 1024 * 1024)
	return LocalDiskSizeGB, nil
}

// GetUsedSpaceGB : retrieves the UsedSpaceGB.
// returns : float64 UsedSpaceGB
// returns : error potential errors
func GetUsedSpaceGB(localMountPoint string) (float64, error) {
	usageStat, err := disk.Usage(localMountPoint)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return -1, fmt.Errorf("failed to get disk utilization for '%v' from host. error: %w", localMountPoint, err)
	}

	// Convert disk usage values to GB
	UsedSpaceGB := float64(usageStat.Used) / (1024 * 1024 * 1024)
	return UsedSpaceGB, nil
}
