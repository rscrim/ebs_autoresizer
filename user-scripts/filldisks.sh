#!/bin/bash

# Ensure parallel is installed
apt-get -y install parallel

# Specify the target directories where the files will be created
# These will be the mountpoints for each EBS volume you want to artificially inflate
target_directories=(
  "/vol/dir1"
  "/vol/dir2"
)

# Function to remove previous .txt files in a directory
remove_previous_files() {
  local directory="$1"
  find "$directory" -type f -name "*.txt" -delete
}

# Function to create files in a directory
create_files() {
  local directory="$1"
  local file_size_gb="$2"
  local num_files="$3"

  for ((i = 1; i <= num_files; i++)); do
    filename="$directory/dummy_file_$i.txt"
    fallocate -l ${file_size_gb}G "$filename"
  done
}

# Set the desired file size (in GB)
file_size_gb=4

# Set the number of files to create in each directory
num_files=20

# Export the create_files function to make it accessible by parallel
export -f create_files

# Iterate over the target directories
for directory in "${target_directories[@]}"; do
  # Remove previous .txt files in the directory
  remove_previous_files "$directory"

  # Create files in parallel using GNU Parallel
  parallel --bar create_files {} $file_size_gb $num_files ::: "$directory"
done
