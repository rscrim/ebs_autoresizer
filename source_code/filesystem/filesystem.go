package filesystem

import (
	"bytes"
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
	// If volumeID starts with "vol-", remove the dash ("-")
	if strings.HasPrefix(volumeID, "vol-") {
		volumeID = strings.Replace(volumeID, "vol-", "vol", 1)
	}

	// Run the "lsblk -o NAME,MOUNTPOINT,SERIAL" command
	cmd := exec.Command("lsblk", "-o", "NAME,MOUNTPOINT,SERIAL")
	fmt.Println("Running command: ", cmd)
	output, err := cmd.Output()
	fmt.Println("Output:", string(output))
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
			} else {
				fmt.Println("Unexpected number of fields in line:", line)
			}
		}
	}

	// The volume ID was not found in the output
	return "", fmt.Errorf("volume ID %s not found", volumeID)
}

// getLocalDeviceName : Retrieves the local NVMe device name for a given mount point.
// mountPoint : string : The local mount point for the volume.
// returns : string : The local NVMe device name or an empty string if not found.
// returns : error : Any error that occurred during the operation.
func getLocalDeviceName(mountPoint string) (string, error) {
	cmd := exec.Command("df", mountPoint)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute 'df' command. error: %w", err)
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected 'df' command output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 6 {
		return "", fmt.Errorf("unexpected 'df' command output")
	}

	deviceName := fields[0] // The device name should be the first field of the second line

	return deviceName, nil
}

// getFileSystemType fetches the file system type of the given mount point.
// mountPoint : string : The mount point whose file system type is required.
// Returns : string : File system type.
// Returns : error : Any error that occurred during operation, nil if operation was successful.
func getFileSystemType(mountPoint string) (string, error) {
	device, err := getLocalDeviceName(mountPoint)
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
// localDeviceName : string : The local device name for the EBS volume
// Returns : error : Any error that occurred during operation, nil if operation was successful.
func ResizeFileSystemByType(filesystem, mountPoint string, localDeviceName string) error {
	var cmd *exec.Cmd
	switch filesystem {
	case "ext4":
		cmd = exec.Command("resize2fs", localDeviceName)
		fmt.Println("Running command: ", cmd)
	case "xfs":
		cmd = exec.Command("xfs_growfs", mountPoint)
		fmt.Println("Running command: ", cmd)
	default:
		return fmt.Errorf("unsupported file system type: %s", filesystem)
	}

	output, err := cmd.CombinedOutput()
	fmt.Println("Output: ", string(output))
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
	localMountPoint, err := GetLocalMountPoint(volume.AWSVolumeID)
	fmt.Println("localMountPoint: ", localMountPoint)
	if err != nil {
		return err
	}

	deviceName, err := getLocalDeviceName(localMountPoint)
	fmt.Println("deviceName: ", deviceName)
	if err != nil {
		return err
	}

	// Get the filesystem type
	filesystem, err := getFileSystemType(localMountPoint)
	fmt.Println("Filesystem: ", filesystem)
	if err != nil {
		return err
	}

	// Resize the filesystem based on its type
	fmt.Println("Attempting to resize the filesystem now!")
	err = ResizeFileSystemByType(filesystem, localMountPoint, deviceName)
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
