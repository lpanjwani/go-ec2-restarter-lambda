package main

import (
	"context"
	"fmt"
	"os"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"net/http"
	"time"
)

type Response struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type WebsiteMonitor struct {
	httpClient *http.Client
	timeout    time.Duration
}

func NewWebsiteMonitor(timeout time.Duration) *WebsiteMonitor {
	return &WebsiteMonitor{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (m *WebsiteMonitor) CheckStatus(websiteURL string) bool {
	resp, err := m.httpClient.Get(websiteURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

type CloudWatchService struct {
	client    *cloudwatch.Client
	namespace string
}

func NewCloudWatchService(cfg aws.Config, namespace string) *CloudWatchService {
	return &CloudWatchService{
		client:    cloudwatch.NewFromConfig(cfg),
		namespace: namespace,
	}
}

func (s *CloudWatchService) PublishMetric(ctx context.Context, metricName string, value float64, dimensions []cwtypes.Dimension) error {
	_, err := s.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(s.namespace),
		MetricData: []cwtypes.MetricDatum{
			{
				MetricName: aws.String(metricName),
				Value:      aws.Float64(value),
				Dimensions: dimensions,
				Unit:       cwtypes.StandardUnitCount,
				Timestamp:  aws.Time(time.Now()),
			},
		},
	})
	return err
}

type EC2Service struct {
	client *ec2.Client
}

func NewEC2Service(cfg aws.Config) *EC2Service {
	return &EC2Service{
		client: ec2.NewFromConfig(cfg),
	}
}

func (s *EC2Service) GetInstanceState(ctx context.Context, instanceID string) (ec2types.InstanceStateName, error) {
	describeResult, err := s.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}
	if len(describeResult.Reservations) == 0 || len(describeResult.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found")
	}
	return describeResult.Reservations[0].Instances[0].State.Name, nil
}

func (s *EC2Service) StopInstance(ctx context.Context, instanceID string) error {
	_, err := s.client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}
	waiter := ec2.NewInstanceStoppedWaiter(s.client)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 300); err != nil {
		return fmt.Errorf("timeout waiting for instance to stop: %w", err)
	}
	return nil
}

func (s *EC2Service) StartInstance(ctx context.Context, instanceID string) error {
	_, err := s.client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}
	return nil
}

func (s *EC2Service) RestartInstance(ctx context.Context, instanceID string) error {
	state, err := s.GetInstanceState(ctx, instanceID)
	if err != nil {
		return err
	}
	switch state {
	case ec2types.InstanceStateNameRunning:
		if err := s.StopInstance(ctx, instanceID); err != nil {
			return err
		}
		state, err = s.GetInstanceState(ctx, instanceID)
		if err != nil {
			return err
		}
	case ec2types.InstanceStateNameStopped:
	default:
		return fmt.Errorf("instance %s is in %s state, waiting for it to stabilize", instanceID, state)
	}
	if state == ec2types.InstanceStateNameStopped {
		if err := s.StartInstance(ctx, instanceID); err != nil {
			return err
		}
	}
	return nil
}

func handler(ctx context.Context) (Response, error) {
	websiteURL := os.Getenv("TF_VAR_website_url")
	if websiteURL == "" {
		return Response{Error: "TF_VAR_website_url environment variable not set"}, fmt.Errorf("TF_VAR_website_url environment variable not set")
	}
	instanceID := os.Getenv("TF_VAR_instance_id")
	if instanceID == "" {
		return Response{Error: "TF_VAR_instance_id environment variable not set"}, fmt.Errorf("TF_VAR_instance_id environment variable not set")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Response{Error: "Failed to load AWS config: " + err.Error()}, err
	}
	monitor := NewWebsiteMonitor(10 * time.Second)
	cwService := NewCloudWatchService(cfg, "WebsiteMonitoring")
	ec2Service := NewEC2Service(cfg)
	isWebsiteUp := monitor.CheckStatus(websiteURL)
	dimensions := []cwtypes.Dimension{
		{
			Name:  aws.String("WebsiteURL"),
			Value: aws.String(websiteURL),
		},
	}
	metricValue := 0.0
	if isWebsiteUp {
		metricValue = 1.0
	}
	if err := cwService.PublishMetric(ctx, "WebsiteAvailability", metricValue, dimensions); err != nil {
		fmt.Printf("Failed to publish CloudWatch metric: %v\n", err)
	}
	if isWebsiteUp {
		return Response{
			Message: fmt.Sprintf("Website %s is up and running. No restart needed.", websiteURL),
		}, nil
	}
	if err := ec2Service.RestartInstance(ctx, instanceID); err != nil {
		return Response{Error: err.Error()}, err
	}
	restartDimensions := []cwtypes.Dimension{
		{
			Name:  aws.String("InstanceID"),
			Value: aws.String(instanceID),
		},
		{
			Name:  aws.String("WebsiteURL"),
			Value: aws.String(websiteURL),
		},
	}
	if err := cwService.PublishMetric(ctx, "InstanceRestart", 1.0, restartDimensions); err != nil {
		fmt.Printf("Failed to publish restart metric: %v\n", err)
	}
	return Response{
		Message: fmt.Sprintf("Website %s was down. Successfully restarted EC2 instance %s", websiteURL, instanceID),
	}, nil
}

func main() {
	lambda.Start(handler)
}