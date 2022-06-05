package cloudmanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"

	awsmap "n8s.io/nebulnetes/pkg/cloudmanager/aws/instancemap"
)

type InstanceStatus int8

const (
	OK InstanceStatus = iota
	SLEEP
	KILLED
)

type CloudManager interface {
	// Build the instance in the selected cloud. Instance type should be a generic term not related to
	// the cloud provider. For instance if a GPU t4 is needed with intel CPU and 32 GB of RAM instead of
	// passing the ec2 name, e.g. `g4dn.8xlarge` the input string should be something like
	// `CPU: IntelXeon, GPU: NvidiaT4, RAM: 32GB`
	BuildInstance(instanceType string) (string, error)
	// Kill the given instance
	KillInstance(instanceId string) error
	// Stop the given instance
	StopInstance(instanceId string) error
	// Reboot the instance
	RebootInstance(instanceId string) error
	// Install in the given instance the kubelet and connect the instance to the kube-cluster
	SetupInstanceAsNode(instanceId string) *v1.Node
	// Check the status of the instance
	CheckInstanceStatus(instanceId string) error
	// Get all instances ID
	GetInstances() []string
}

func ParseInstanceType(instanceType string) map[string]string {
	hardwareMap := make(map[string]string)
	for _, subString := range strings.Split(instanceType, ", ") {
		if strings.Contains(subString, "CPU:") {
			hardwareMap["CPU"] = strings.Replace(subString, "CPU: ", "", 1)
		} else if strings.Contains(subString, "GPU:") {
			hardwareMap["GPU"] = strings.Replace(subString, "GPU: ", "", 1)
		} else if strings.Contains(subString, "RAM:") {
			hardwareMap["RAM"] = strings.Replace(subString, "RAM: ", "", 1)
		} else {
			log.Println("Found an extra line by the instanceType parser: ", subString)
		}
	}
	return hardwareMap
}

// Get the instance name and the OS type from the hardware info.
// Note that if instances are not found for neither the GPU and CPU
// specified a tuple of empty strings is returned.
func GetAWSInstance(hardwareInfo map[string]string) (string, string) {
	var instanceName string
	var instanceOS string
	instanceMapGPU := awsmap.NewGpu2InstanceMap(hardwareInfo["GPU"])
	if instanceMapGPU != nil {
		for key, value := range *(*instanceMapGPU).GetInstances() {
			if value.HasCpu(hardwareInfo["CPU"]) && value.HasRam(hardwareInfo["RAM"]) {
				instanceName = key
				break
			}
		}
		instanceOS = awsmap.DEFAULT_GPU_OS
	} else {
		// we provide an instance with the given CPU and no GPU
		instanceMapCPU := awsmap.NewCpu2InstanceMap(hardwareInfo["CPU"])
		if instanceMapCPU != nil {
			instanceOS = awsmap.DEFAULT_CPU_OS
			for key, value := range *instanceMapCPU.GetInstances() {
				if value.HasRam(hardwareInfo["RAM"]) {
					instanceName = key
					break
				}
			}
		}
	}

	return instanceName, instanceOS
}

func NewAWSDefaultClient() *ec2.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("Error while loading the AWS client with Default Configuration. Got error " + err.Error())
	}
	client := ec2.NewFromConfig(cfg)
	return client
}

type AWSCloudManager struct {
	client    *ec2.Client
	instances *map[string]InstanceStatus
}

func NewAWSCloudManager() *AWSCloudManager {
	instances := make(map[string]InstanceStatus)
	cm := AWSCloudManager{
		client:    NewAWSDefaultClient(),
		instances: &instances,
	}
	return &cm
}

// Build an AWS instance
func (cm *AWSCloudManager) BuildInstance(instanceType string) (string, error) {
	hardwareMap := ParseInstanceType(instanceType)
	instanceName, instanceOS := GetAWSInstance(hardwareMap)
	if instanceName == "" || instanceOS == "" {
		errorMsg := fmt.Sprintln("No Instance satisfying the given requirements has been found. Given requirements:", instanceType)
		return "", errors.New(errorMsg)
	}
	tag_name := "Name"
	tag_value := fmt.Sprint("N8s Node: ", instanceName)

	var countNum int32 = 1
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(instanceOS),
		InstanceType: types.InstanceType(instanceName),
		MinCount:     &countNum,
		MaxCount:     &countNum,
	}

	result, err := cm.client.RunInstances(context.TODO(), input)

	if err != nil {
		errorMsg := fmt.Sprintln("Got an error creating an instance:", err)
		return "", errors.New(errorMsg)
	}

	tagInput := &ec2.CreateTagsInput{
		Resources: []string{*result.Instances[0].InstanceId},
		Tags: []types.Tag{
			{
				Key:   &tag_name,
				Value: &tag_value,
			},
		},
	}

	_, err = cm.client.CreateTags(context.TODO(), tagInput)
	if err != nil {
		errorMsg := fmt.Sprintln("Got an error tagging the instance:", err)
		return "", errors.New(errorMsg)
	}
	instanceId := result.Instances[0].InstanceId
	(*cm.instances)[*instanceId] = OK
	return *result.Instances[0].InstanceId, nil
}

func (cm *AWSCloudManager) StopInstance(instanceId string) error {
	dryRunFlag := true
	input := &ec2.StopInstancesInput{
		InstanceIds: []string{
			instanceId,
		},
		DryRun: &dryRunFlag,
	}

	_, err := cm.client.StopInstances(context.TODO(), input)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "DryRunOperation" {
		fmt.Println("User has permission to stop instances.")
		dryRunFlag = false
		input.DryRun = &dryRunFlag
		_, err = cm.client.StopInstances(context.TODO(), input)
	}

	if err != nil {
		fmt.Println("Got an error stopping the instance")
		fmt.Println(err)
		return err
	}

	fmt.Println("Stopped instance with ID " + instanceId)
	(*cm.instances)[instanceId] = SLEEP
	return nil
}

func (cm *AWSCloudManager) RestartInstance(instanceId string) error {
	if (*cm.instances)[instanceId] != SLEEP {
		errorMessage := fmt.Sprint("The linked instances isn't in the SLEEP model. Got ", (*cm.instances)[instanceId])
		return errors.New(errorMessage)
	}

	dryRunFlag := true
	input := &ec2.StartInstancesInput{
		InstanceIds: []string{
			instanceId,
		},
		DryRun: &dryRunFlag,
	}

	_, err := cm.client.StartInstances(context.TODO(), input)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "DryRunOperation" {
		fmt.Println("User has permission to start an instance.")
		dryRunFlag = false
		input.DryRun = &dryRunFlag
		_, err = cm.client.StartInstances(context.TODO(), input)
	}
	if err != nil {
		fmt.Println("Got an error starting the instance")
		return err
	}

	(*cm.instances)[instanceId] = OK
	return nil
}

func (cm *AWSCloudManager) RebootInstance(instanceId string) error {
	if (*cm.instances)[instanceId] != OK {
		errorMsg := fmt.Sprint("The linked instance cannot be rebooted since it is not active. Actual status: ", (*cm.instances)[instanceId])
		return errors.New(errorMsg)
	}
	dryRunFlag := true
	input := &ec2.RebootInstancesInput{
		InstanceIds: []string{
			instanceId,
		},
		DryRun: &dryRunFlag,
	}

	_, err := cm.client.RebootInstances(context.TODO(), input)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "DryRunOperation" {
		fmt.Println("User has permission to enable monitoring.")
		dryRunFlag = false
		input.DryRun = &dryRunFlag
		_, err = cm.client.RebootInstances(context.TODO(), input)
	}

	if err != nil {
		fmt.Println("Got an error rebooting the instance")
		return err
	}

	fmt.Println("Rebooted instance with ID " + instanceId)
	return nil
}

func (cm *AWSCloudManager) KillInstance(instanceId string) error {
	dryRunFlag := true
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{
			instanceId,
		},
		DryRun: &dryRunFlag,
	}

	_, err := cm.client.TerminateInstances(context.TODO(), input)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "DryRunOperation" {
		fmt.Println("User has permission to enable monitoring.")
		dryRunFlag = false
		input.DryRun = &dryRunFlag
		_, err = cm.client.TerminateInstances(context.TODO(), input)
	}

	if err != nil {
		fmt.Println("Got an error terminating the instance")
		return err
	}

	fmt.Println("Terminated instance with ID " + instanceId)
	(*cm.instances)[instanceId] = KILLED
	return nil
}

func (cm *AWSCloudManager) CheckInstanceStatus(instanceId string) error {
	status := (*cm.instances)[instanceId]
	if status != OK {
		errorMsg := fmt.Sprint("Instance ", instanceId, " is not running. Got status ", status)
		return errors.New(errorMsg)
	}
	return nil
}

func (cm *AWSCloudManager) GetInstances() *[]string {
	instances := make([]string, 0)
	for instanceId, status := range *cm.instances {
		if status == OK {
			instances = append(instances, instanceId)
		}
	}
	return &instances
}

// TODO: Add Key-pair security to instances
func (cm *AWSCloudManager) GetSSHClientConfig() *ssh.ClientConfig {
	newClientConfig := ssh.ClientConfig{
		User: "user",
		Auth: []ssh.AuthMethod{},
	}
	return &newClientConfig
}

func (cm *AWSCloudManager) GetHostname(instanceId string) (string, error) {
	// Use the idms client for exploring the instance Metadata
	instanceDescriptionInput := ec2.DescribeInstancesInput{
		InstanceIds: []string{
			instanceId,
		},
	}
	instanceDescriptionOutput, err := cm.client.DescribeInstances(
		context.TODO(), &instanceDescriptionInput,
	)
	if err != nil {
		log.Printf("error: %v", err)
		return "", err
	}
	instance := instanceDescriptionOutput.Reservations[0].Instances[0]
	ip := instance.PublicIpAddress
	return *ip, nil
}

func GetAWSNodeSetup() []string {
	cmd := make([]string, 0)
	cmd = append(cmd, "")
	return cmd
}

func (cm *AWSCloudManager) SetupInstanceAsNode(instanceId string) *v1.Node {
	config := cm.GetSSHClientConfig()
	hostname, err := cm.GetHostname(instanceId)
	if err != nil {
		log.Print("Error while retrieving the hostname for ", instanceId, ". Got error ", err)
		return nil
	}
	conn, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		log.Print("Error while connecting to ", hostname, ". Got error ", err)
		return nil
	}

	defer conn.Close()
	session, err := conn.NewSession()

	if err != nil {
		log.Fatalf("unable to create session: %s", err)
	}
	defer session.Close()
	//sshSession = ssh.New()
	cmds := GetAWSNodeSetup()
	stdinBuf, _ := session.StdinPipe()
	err = session.Shell()
	if err != nil {
		log.Print("Error while instantiating a shell on ", instanceId, ". Got error ", err)
	}
	for _, cmd := range cmds {
		stdinBuf.Write([]byte(cmd))
	}
	// TODO: Understand how nodes behaves and how they are created in order to return the Node object
	newNode := v1.Node{}
	return &newNode
}
