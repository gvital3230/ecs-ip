package aws

import (
	"context"
	"fmt"
	"log"

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

func NewStore() *Store {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	return &Store{
		ecsClient: ecs.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
	}
}

func (store *Store) Clusters() []Cluster {
	// get list of GetClusters Arn
	res := []Cluster{}
	clusterList, err := store.ecsClient.ListClusters(context.TODO(), &ecs.ListClustersInput{})
	if err != nil {
		log.Fatal(err)
	}

	// get list of full clusters data
	clusterDescriptions, err := store.ecsClient.DescribeClusters(context.TODO(), &ecs.DescribeClustersInput{
		Clusters: clusterList.ClusterArns,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, clusterDescription := range clusterDescriptions.Clusters {
		cl := Cluster{
			Arn:  *clusterDescription.ClusterArn,
			Name: *clusterDescription.ClusterName,
		}
		cl.Services = store.services(clusterDescription)
		res = append(res, cl)
	}
	return res
}

func (s *Store) services(c ecsTypes.Cluster) []Service {
	var res []Service

	servicesList, err := s.ecsClient.ListServices(context.TODO(), &ecs.ListServicesInput{
		Cluster: c.ClusterArn,
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, sArn := range servicesList.ServiceArns {
		servicesDescriptions, _ := s.ecsClient.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
			Cluster:  c.ClusterArn,
			Services: []string{sArn},
		})
		for _, serviceDescription := range servicesDescriptions.Services {

			instances, err := s.serviceInstances(c.ClusterArn, serviceDescription)
			if err != nil {
				log.Fatal(err)
			}

			s := Service{
				Name:  *serviceDescription.ServiceName,
				Image: s.appImage(serviceDescription),
				PrivateIPs: lo.Map(instances, func(instance ec2Types.Instance, _ int) string {
					return *instance.PrivateIpAddress
				}),
				PublicIPs: lo.Map(instances, func(instance ec2Types.Instance, _ int) string {
					return *instance.PublicIpAddress
				}),
			}
			res = append(res, s)
		}
	}
	return res
}

func (s *Store) appImage(service ecsTypes.Service) string {
	td := service.TaskDefinition

	taskDefinition, err := s.ecsClient.DescribeTaskDefinition(context.TODO(), &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: td,
	})
	if err != nil {
		log.Fatal(err)
	}
	return *taskDefinition.TaskDefinition.ContainerDefinitions[0].Image
}

func (s *Store) serviceInstances(clusterArn *string, service ecsTypes.Service) ([]ec2Types.Instance, error) {
	// List the tasks running in the service
	listTasksOutput, err := s.ecsClient.ListTasks(context.TODO(), &ecs.ListTasksInput{
		Cluster:     clusterArn,
		ServiceName: service.ServiceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	if len(listTasksOutput.TaskArns) == 0 {
		log.Println("No tasks found")
		return nil, nil
	}

	// Describe the tasks to get the container instances
	describeTasksOutput, err := s.ecsClient.DescribeTasks(context.TODO(), &ecs.DescribeTasksInput{
		Cluster: clusterArn,
		Tasks:   listTasksOutput.TaskArns,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe tasks: %w", err)
	}

	var containerInstanceArns []string
	for _, task := range describeTasksOutput.Tasks {
		if task.ContainerInstanceArn != nil {
			containerInstanceArns = append(containerInstanceArns, *task.ContainerInstanceArn)
		}
	}

	if len(containerInstanceArns) == 0 {
		log.Println("No container instances found")
		return nil, nil
	}

	// Describe container instances to get the EC2 instance IDs
	describeContainerInstancesOutput, err := s.ecsClient.DescribeContainerInstances(context.TODO(), &ecs.DescribeContainerInstancesInput{
		Cluster:            clusterArn,
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
	describeInstancesOutput, err := s.ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
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
