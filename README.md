# Go EC2 Restarter Lambda

## Environment Variables

This project uses a unified approach for environment variables shared between the Go application and Terraform deployment.

1. Copy the example environment file:

    ```
    cp .env.example .env
    ```

2. Edit the `.env` file with your specific values:

    ```
    # AWS Configuration
    TF_VAR_aws_region=us-east-1

    # Lambda Configuration
    TF_VAR_lambda_function_name=mete-wordpress-ec2-restarter

    # Application Configuration
    TF_VAR_website_url=https://your-website.com
    TF_VAR_instance_id=i-1234567890abcdef0
    ```

3. The deployment script will automatically use these variables for both the Go application and Terraform.

## Deployment Instructions

### Prerequisites

1. Install [Go](https://golang.org/doc/install)
2. Install [Terraform](https://learn.hashicorp.com/tutorials/terraform/install-cli)
3. Configure AWS credentials (`aws configure`)

### Deployment Steps

1. Clone this repository
2. Set up your environment variables in the `.env` file as described above
3. Make the deployment script executable:
    ```
    chmod +x deploy.sh
    ```
4. Run the deployment script:
    ```
    ./deploy.sh
    ```

## Cleanup

To remove all resources created by this deployment:

```
cd deploy
terraform destroy
```
