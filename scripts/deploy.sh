#!/bin/bash
set -e

# Check if .env file exists
if [ -f .env ]; then
  echo "Loading environment variables from .env file..."

  # Read each line from .env file
  while IFS= read -r line || [ -n "$line" ]; do
    # Skip comments and empty lines
    [[ $line =~ ^#.*$ || -z $line ]] && continue

    # Extract variable name and value
    key=$(echo "$line" | cut -d= -f1)
    value=$(echo "$line" | cut -d= -f2-)

    # Export as regular environment variable
    export "$key"="$value"

    # Also export with TF_VAR_ prefix for Terraform
    export "TF_VAR_${key}"="$value"
  done < .env
else
  echo "No .env file found. Please create one based on .env.example"
  exit 1
fi

# Validate required environment variables
if [ -z "$TF_VAR_website_url" ] || [ -z "$TF_VAR_instance_id" ]; then
  echo "Error: TF_VAR_website_url and TF_VAR_instance_id must be set in the .env file"
  exit 1
fi

echo "Building Go Lambda function..."
mkdir -p build
GOOS=linux GOARCH=amd64 go build -o build/main main.go

echo "Initializing Terraform..."
cd infra

echo "Planning Terraform deployment..."
terraform plan -out=tfplan

echo "Applying Terraform changes..."
terraform apply tfplan

echo "Deployment complete!"
