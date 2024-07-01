package aws

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/samber/lo"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecsTypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Store struct {
	ecsClient *ecs.Client
	ec2Client *ec2.Client
}

func NewStore(region string) *Store {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}

	return &Store{
		ecsClient: ecs.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
	}
}

func (store *Store) Clusters() []Cluster {
	// get list of Clusters ARNs
	res := []Cluster{}
	list, err := store.ecsClient.ListClusters(context.TODO(), &ecs.ListClustersInput{})
	if err != nil {
		log.Fatal(err)
	}

	// get list of full details data, ignore pagination for now, we have less than 100 details
	details, err := store.ecsClient.DescribeClusters(context.TODO(), &ecs.DescribeClustersInput{
		Clusters: list.ClusterArns,
	})
	if err != nil {
		log.Fatal(err)
	}

	// set up orchestrator primitives to fetch cluster details concurrently
	var wg sync.WaitGroup
	ch := make(chan Cluster, len(details.Clusters))

	// fetch cluster details concurrently
	for _, cluster := range details.Clusters {
		wg.Add(1)
		go store.clusterDetails(cluster, &wg, ch)
	}

	// close channel when all clusters are fetched
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect clusters from channel
	for cl := range ch {
		res = append(res, cl)
	}

	return res
}

func (store *Store) clusterDetails(cl ecsTypes.Cluster, wg *sync.WaitGroup, ch chan Cluster) {
	defer wg.Done()

	ch <- Cluster{
		Arn:      *cl.ClusterArn,
		Name:     *cl.ClusterName,
		Services: store.services(cl),
	}
}

func (store *Store) services(c ecsTypes.Cluster) []Service {
	var res []Service

	// get list of services Arn in the cluster
	list, err := store.ecsClient.ListServices(context.TODO(), &ecs.ListServicesInput{
		Cluster: c.ClusterArn,
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(list.ServiceArns) == 0 {
		return res
	}

	// get full services details, ignore pagination for now, we have less than 100 services
	details, err := store.ecsClient.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
		Cluster:  c.ClusterArn,
		Services: list.ServiceArns,
	})
	if err != nil {
		log.Fatal(err)
	}

	// set up orchestrator primitives to fetch service details concurrently
	var wg sync.WaitGroup
	ch := make(chan Service, len(details.Services))

	for _, service := range details.Services {
		wg.Add(1)
		// fetch service details concurrently
		go store.serviceDetails(service, &wg, ch)
	}

	// close channel when all services are fetched
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect services from channel
	for service := range ch {
		res = append(res, extractMetadata(service))
	}

	return res
}

func extractMetadata(service Service) Service {
	// try to find labels in the image name
	parts := strings.Split(service.Image, ":")
	if len(parts) != 0 {
		service.Version = parts[1]
	}

	// then take remaining part of the image name and try to extract some metadata
	parts = strings.Split(parts[0], "/")

	if len(parts) != 0 {
		image := parts[len(parts)-1]
		remaining := ""
		// remove non meaminful prefixes
		_, remaining = findAndCut(image, []string{"ptah"})

		// find known env names
		knownEnvs := []string{"dev2", "dev", "development", "stage", "staging", "prod", "ci"}
		service.Env, remaining = findAndCut(remaining, knownEnvs)
		if service.Env == "" {
			service.Env, _ = findAndCut(service.Version, knownEnvs)
		}

		service.App, remaining = findAndCut(remaining, []string{"wp-multisite", "wl-widgets", "social-auth", "wp-wl-elementor", "wl-messenger", "wl-fitbuilder", "wl-explorer"})
		service.Component = strings.Trim(remaining, "-")
	}
	return service
}

func findAndCut(input string, values []string) (string, string) {
	for _, value := range values {
		// Create a regex pattern to match the value as a whole word
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(value))
		re := regexp.MustCompile(pattern)

		if re.FindStringIndex(input) != nil {
			// Remove all occurrences of the found value as a whole word
			cutString := re.ReplaceAllString(input, "")
			return value, cutString
		}
	}
	return "", input
	// for _, value := range values {
	// 	if index := strings.Index(input, value); index != -1 {
	// 		// Remove all occurrences of the found value
	// 		cutString := strings.ReplaceAll(input, value, "")
	// 		return value, cutString
	// 	}
	// }
	// return "", input
}

func (store *Store) serviceDetails(service ecsTypes.Service, wg *sync.WaitGroup, ch chan Service) {
	defer wg.Done()
	instances, err := store.serviceInstances(service)
	if err != nil {
		log.Fatal(err)
	}
	ch <- Service{
		Name:  *service.ServiceName,
		Image: store.appImage(service),
		PrivateIPs: lo.Map(instances, func(instance ec2Types.Instance, _ int) string {
			return *instance.PrivateIpAddress
		}),
		PublicIPs: lo.Map(instances, func(instance ec2Types.Instance, _ int) string {
			return *instance.PublicIpAddress
		}),
	}
}

func (store *Store) appImage(service ecsTypes.Service) string {
	taskDefinition, err := store.ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: service.TaskDefinition,
	})
	if err != nil {
		log.Fatal(err)
	}

	// we assume that there is only one container in the task definition or at least the first one is the one we are interested in
	return *taskDefinition.TaskDefinition.ContainerDefinitions[0].Image
}

func (store *Store) serviceInstances(service ecsTypes.Service) ([]ec2Types.Instance, error) {
	// List the tasks running in the service
	taskList, err := store.ecsClient.ListTasks(context.TODO(), &ecs.ListTasksInput{
		Cluster:     service.ClusterArn,
		ServiceName: service.ServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	if len(taskList.TaskArns) == 0 {
		log.Println("No tasks found")
		return nil, nil
	}

	// Describe the tasks to get the container instances
	taskDetails, err := store.ecsClient.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
		Cluster: service.ClusterArn,
		Tasks:   taskList.TaskArns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe tasks for cluster %v: %w", *service.ClusterArn, err)
	}

	var containerInstanceArns []string
	for _, task := range taskDetails.Tasks {
		if task.ContainerInstanceArn != nil {
			containerInstanceArns = append(containerInstanceArns, *task.ContainerInstanceArn)
		}
	}

	if len(containerInstanceArns) == 0 {
		log.Println("No container instances found")
		return nil, nil
	}

	// Describe container instances to get the EC2 instance IDs
	describeContainerInstancesOutput, err := store.ecsClient.DescribeContainerInstances(context.TODO(), &ecs.DescribeContainerInstancesInput{
		Cluster:            service.ClusterArn,
		ContainerInstances: containerInstanceArns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe container instances: %w", err)
	}

	var ec2InstanceIds []string
	for _, containerInstance := range describeContainerInstancesOutput.ContainerInstances {
		if containerInstance.Ec2InstanceId != nil {
			ec2InstanceIds = append(ec2InstanceIds, *containerInstance.Ec2InstanceId)
		}
	}

	if len(ec2InstanceIds) == 0 {
		fmt.Println("No EC2 instances found")
		return nil, nil
	}

	// Describe EC2 instances to get their IP addresses
	describeInstancesOutput, err := store.ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: ec2InstanceIds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	// Get Instances
	res := []ec2Types.Instance{}
	for _, reservation := range describeInstancesOutput.Reservations {
		res = append(res, reservation.Instances...)
	}
	return res, nil
}
