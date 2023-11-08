import os
import subprocess
import shutil
from tqdm import tqdm


def extract_version_from_control():
    control_file = "/Users/username/rscrim/ebs_autoresizer/build/DEBIAN/control"
    with open(control_file, "r") as f:
        for line in f:
            if line.startswith("Version:"):
                return line.split(":")[1].strip()
    return None


# Set the environment variables
os.environ["GOOS"] = "linux"
os.environ["GOARCH"] = "amd64"

# Update the directory references
source_code_dir = "/Users/username/rscrim/ebs_autoresizer/ebsmon"
destination_bin_dir = "/Users/username/rscrim/ebs_autoresizer/build/usr/local/bin"
package_dir = "/Users/username/rscrim/ebs_autoresizer/build"
user_scripts_dir = "/Users/username/rscrim/ebs_autoresizer/user-scripts"

test_config = os.path.join(user_scripts_dir, "sample_config.yaml")
filldisks = os.path.join(user_scripts_dir, "filldisks.sh")

# Servers
servers = ["serverA", "serverB"]

# Remove created files
print("Cleaning up...")
ebsmon_path = os.path.join(destination_bin_dir, "ebsmon")
deb_path = os.path.join(package_dir, "ebsmon.deb")

if os.path.isfile(ebsmon_path):
    os.remove(ebsmon_path)

if os.path.isfile(deb_path):
    os.remove(deb_path)

version = extract_version_from_control()
if version:
    # Build the local package
    print("Building the Go package...")
    build_cmd = f"go build -ldflags '-X main.version={version}' -o ebsmon"
    subprocess.run(build_cmd, shell=True, check=True, cwd=source_code_dir)
else:
    print("Failed to extract version from control file.")
    exit(1)

# Move the built Go binary to the appropriate directory
print("Moving the built binary...")
shutil.move(os.path.join(source_code_dir, "ebsmon"), destination_bin_dir)

# Build the package using dpkg-deb
print("Building the .deb package...")
build_package_cmd = ["dpkg-deb", "--build", ".", "ebsmon.deb"]
subprocess.check_call(build_package_cmd, cwd=package_dir)

# SCP the files to the servers
for server in tqdm(
    servers, desc="Copying files"
):  # Wrap servers with tqdm for progress bar
    print(f"Copying the files to the server {server}...")
    # Perform SCP for ebsmon.deb
    scp_package_cmd = f"scp {package_dir}/ebsmon.deb {server}:~/ebsmon"
    subprocess.run(scp_package_cmd, shell=True, check=True)
    # Perform SCP for config.yaml
    scp_config_cmd = f"scp {test_config} {server}:~/ebsmon"
    subprocess.run(scp_config_cmd, shell=True, check=True)
    # Perform SCP for filldisk.sh
    scp_filldisk_cmd = f"scp {filldisks} {server}:~/ebsmon"
    subprocess.run(scp_filldisk_cmd, shell=True, check=True)

print("Package built and SCP completed successfully.")
