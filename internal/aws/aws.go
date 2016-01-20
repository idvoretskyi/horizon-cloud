package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AWS struct {
	EC2 *ec2.EC2
}

func New() *AWS {
	return &AWS{
		EC2: ec2.New(session.New(&aws.Config{Region: aws.String("us-east-1")})),
	}
}

func (a *AWS) ListServers() ([]*Server, error) {
	resp, err := a.EC2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:fusion-cluster"),
				Values: []*string{aws.String("psuedocluster")}, // TODO
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if resp.NextToken != nil {
		return nil, errors.New("paginated DescribeInstances response not supported yet")
	}

	out := []*Server{}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			out = append(out, ec2InstanceToServer(instance))
		}
	}

	return out, nil
}

func ec2InstanceToServer(i *ec2.Instance) *Server {
	srv := &Server{
		InstanceID: *i.InstanceId,

		Started:      *i.LaunchTime,
		State:        *i.State.Name,
		InstanceType: *i.InstanceType,
	}

	if i.StateReason != nil && i.StateReason.Message != nil {
		srv.StateReason = *i.StateReason.Message
	}
	if i.PublicIpAddress != nil {
		srv.PublicIP = *i.PublicIpAddress
	}
	if i.PrivateIpAddress != nil {
		srv.PrivateIP = *i.PrivateIpAddress
	}

	return srv
}

func (a *AWS) StatServer(id string) (*Server, error) {
	resp, err := a.EC2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-id"),
				Values: []*string{&id},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if resp.NextToken != nil {
		// RSI: serious log for seriousness
		return nil, errors.New("paginated DescribeInstances response should not happen")
	}

	var out *Server

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			if out != nil {
				return nil, errors.New("got more than one instance")
			}

			out = ec2InstanceToServer(instance)
		}
	}

	return out, nil
}

func (a *AWS) StartServer(instancetype string) (*Server, error) {
	log.Printf("Starting instance (type = %v)", instancetype)
	runResp, err := a.EC2.RunInstances(&ec2.RunInstancesInput{
		InstanceType: &instancetype,
		KeyName:      aws.String("fusiondev"),    // RSI
		ImageId:      aws.String("ami-fc5e7594"), // TODO: lookup table
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		// TODO: VPC
		// TODO: appropriate security group
	})
	if err != nil {
		return nil, err
	}

	var srv *Server

	for _, instance := range runResp.Instances {
		if srv != nil {
			return nil, errors.New("got more than one instance")
		}

		srv = ec2InstanceToServer(instance)
	}

	log.Printf("Got new instance %#v", srv)

	// tag it as being in the fusion-cluster

	_, err = a.EC2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(srv.InstanceID)},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("fusion-cluster"),
				Value: aws.String("psuedocluster"), // TODO
			},
		},
	})
	if err != nil {
		// RSI: log serious, we may be leaking instances here
		return nil, err
	}

	// now, wait for it to start or die early

	started := time.Now()
	for time.Now().Sub(started) < time.Minute*5 {
		time.Sleep(5 * time.Second)

		newSrv, err := a.StatServer(srv.InstanceID)
		if err != nil {
			log.Printf("Couldn't Stat instance recently started: %v", err)
			continue
		}

		srv = newSrv

		log.Printf("state poll showed %s", srv.State)

		switch srv.State {
		case "running":
			return srv, nil
		case "shutting-down", "terminated", "stopping":
			return nil, fmt.Errorf("Instance %v transitioned to state %v unexpectedly",
				srv.InstanceID, srv.State)
		case "pending":
			// do nothing
		default:
			// RSI: log serious
			log.Printf("Unknown server state %v", srv.State)
		}
	}

	// timed out, kill the instance and return an error

	err = a.StopServer(srv.InstanceID)
	if err != nil {
		// TODO: log serious
	}

	return nil, errors.New("Instance did not start in 5 minutes")
}

func (a *AWS) StopServer(instanceid string) error {
	_, err := a.EC2.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instanceid},
	})

	// TODO: consider validating response

	return err
}

type Server struct {
	InstanceID string

	Started      time.Time
	State        string
	StateReason  string
	InstanceType string
	PublicIP     string
	PrivateIP    string
}
