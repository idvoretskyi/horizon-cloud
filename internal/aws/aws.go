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

type Server struct {
	InstanceID string

	Started      time.Time
	State        string
	StateReason  string
	InstanceType string
	PublicIP     string
	PrivateIP    string
}

type Volume struct {
	ID   string
	Size int64
}

type AWS struct {
	EC2     *ec2.EC2
	Cluster string
}

func New(cluster string) *AWS {
	return &AWS{
		EC2:     ec2.New(session.New(&aws.Config{Region: aws.String("us-west-1")})),
		Cluster: cluster,
	}
}

func (a *AWS) WaitVolume(id string) error {
	// RSI: should exit immediately if volume doesn't exist.
	started := time.Now()
	delay := time.Second
	for time.Now().Sub(started) < time.Minute*5 {
		time.Sleep(delay)
		delay += time.Second
		if delay > time.Second*5 {
			delay = time.Second * 5
		}

		ready, err := a.VolumeReady(id)
		if err != nil {
			log.Printf("Couldn't check volume readiness: %v", err)
			continue
		}
		if ready {
			return nil
		}
	}

	return errors.New("EC2 volume did not become ready in 5 minutes")
}

const GP2 = "gp2"
const IO1 = "io1"

func (a *AWS) CreateVolume(size int64, volType string) (*Volume, error) {
	az := "us-west-1a"
	vol, err := a.EC2.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: &az,
		Size:             &size,
		VolumeType:       &volType,
	})
	if err != nil {
		return nil, err
	}
	if vol.VolumeId == nil || vol.Size == nil {
		return nil, fmt.Errorf("bad volume: %s", vol)
	}
	err = a.WaitVolume(*vol.VolumeId)
	if err != nil {
		return nil, err
	}
	log.Printf("created volume %s", *vol.VolumeId)
	return &Volume{
		ID:   *vol.VolumeId,
		Size: *vol.Size,
	}, nil
}

func (a *AWS) DeleteVolume(id string) error {
	// Returns an empty struct.
	_, err := a.EC2.DeleteVolume(&ec2.DeleteVolumeInput{VolumeId: &id})
	if err != nil {
		return err
	}
	log.Printf("deleted volume %s", id)
	return nil
}

func (a *AWS) ListServers() ([]*Server, error) {
	resp, err := a.EC2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:fusion-cluster"),
				Values: []*string{aws.String(a.Cluster)},
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

func (a *AWS) waitServer(instanceID string,
	fn func(*Server) (bool, error)) (*Server, error) {

	var srv *Server

	started := time.Now()
	delay := time.Second
	for time.Now().Sub(started) < time.Minute*5 {
		time.Sleep(delay)
		delay += time.Second
		if delay > time.Second*5 {
			delay = time.Second * 5
		}

		newSrv, err := a.StatServer(instanceID)
		if err != nil {
			log.Printf("Couldn't Stat instance: %v", err)
			continue
		}

		srv = newSrv

		done, err := fn(srv)
		if err != nil {
			return nil, err
		}
		if done {
			return srv, nil
		}
	}

	return nil, errors.New("Instance did not transition in 5 minutes")
}

func (a *AWS) VolumeReady(id string) (bool, error) {
	resp, err := a.EC2.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIds: []*string{&id},
	})
	if err != nil {
		return false, err
	}
	if len(resp.Volumes) != 1 {
		return false, fmt.Errorf("expected 1 volume but got %d", len(resp.Volumes))
	}
	vol := resp.Volumes[0]
	switch *vol.State {
	case ec2.VolumeStateCreating:
		return false, nil
	case ec2.VolumeStateAvailable:
		return true, nil
	case ec2.VolumeStateInUse, ec2.VolumeStateDeleting,
		ec2.VolumeStateDeleted, ec2.VolumeStateError:
		// RSI: log a serious error (unusable volume state)
		return false, fmt.Errorf("unusable ec2 volume state '%s'", vol.State)
	default:
		// RSI: log a serious error (API changes)
		return false, fmt.Errorf("unexpected ec2 volume state '%s'", vol.State)
	}
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

func (a *AWS) StartServer(instancetype string, ami string) (*Server, error) {
	log.Printf("Starting instance (type = %v)", instancetype)

	runResp, err := a.EC2.RunInstances(&ec2.RunInstancesInput{
		InstanceType: &instancetype,
		KeyName:      aws.String("fusiondev"), // RSI
		ImageId:      aws.String(ami),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		SecurityGroupIds: []*string{ // TODO
			aws.String("sg-a7be25de"), // ssh_ping
			aws.String("sg-a56810dc"), // http(s)
		},
		SubnetId: aws.String("subnet-4dc43067"),
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

	// If we're too fast, the server won't show up as existing by the time we
	// call CreateTags. This call waits until the server shows up in the AWS
	// metadata store so that we can tag it.
	_, err = a.waitServer(srv.InstanceID, func(*Server) (bool, error) {
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// tag it as being in the fusion-cluster
	_, err = a.EC2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(srv.InstanceID)},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("fusion-cluster"),
				Value: aws.String(a.Cluster),
			},
		},
	})
	if err != nil {
		// RSI: log serious, we may be leaking instances here
		return nil, err
	}

	// now, wait for it to start or die early

	srv, err = a.waitServer(srv.InstanceID, func(srv *Server) (bool, error) {
		switch srv.State {
		case "running":
			return true, nil
		case "shutting-down", "terminated", "stopping":
			return false, fmt.Errorf("Instance %v transitioned to state %v unexpectedly",
				srv.InstanceID, srv.State)
		case "pending":
			// do nothing
		default:
			// RSI: log serious
			log.Printf("Unknown server state %v", srv.State)
		}
		return false, nil
	})

	if err != nil {
		// kill the instance and return the original error

		_ = a.TerminateServer(srv.InstanceID)
		// TODO: check for error and log serious

		return nil, err
	}

	return srv, nil
}

func (a *AWS) TerminateServer(instanceID string) error {
	_, err := a.EC2.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instanceID},
	})
	if err != nil {
		return err
	}

	// TODO: consider validating response

	_, err = a.waitServer(instanceID, func(srv *Server) (bool, error) {
		switch srv.State {
		case "pending", "running", "shutting-down", "stopping":
			return false, nil
		case "terminated":
			return true, nil
		case "stopped":
			return false, errors.New("server transitioned to stopped after terminate call")
		default:
			log.Printf("Unknown server state %v", srv.State)
			return false, nil
		}
	})

	return err
}

func (a *AWS) StopServer(instanceID string) error {
	_, err := a.EC2.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{&instanceID},
	})
	if err != nil {
		return err
	}

	// TODO: consider validating response

	_, err = a.waitServer(instanceID, func(srv *Server) (bool, error) {
		switch srv.State {
		case "pending", "running", "shutting-down", "stopping":
			return false, nil
		case "terminated":
			return false, errors.New("server transitioned to terminated after stop call")
		case "stopped":
			return true, nil
		default:
			log.Printf("Unknown server state %v", srv.State)
			return false, nil
		}
	})

	return err
}

func (a *AWS) CreateImage(instanceID string, imageName string) (string, error) {
	resp, err := a.EC2.CreateImage(&ec2.CreateImageInput{
		InstanceId: aws.String(instanceID),
		Name:       aws.String(imageName),
	})
	if err != nil {
		return "", err
	}

	ami := *resp.ImageId

	started := time.Now()
	for time.Now().Sub(started) < time.Minute*15 {
		time.Sleep(5 * time.Second)

		resp, err := a.EC2.DescribeImages(&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(ami)},
		})
		if err != nil {
			log.Printf("Couldn't describe image recently created: %v", err)
			continue
		}

		if len(resp.Images) != 1 {
			return "", fmt.Errorf("DescribeImagesOutput returned %v images, wanted 1",
				len(resp.Images))
		}

		image := *resp.Images[0]
		state := *image.State

		switch state {
		case "pending":
			continue
		case "available":
			return ami, nil
		case "invalid", "deregistered", "transient", "failed", "error":
			var reason string
			if image.StateReason != nil && image.StateReason.Message != nil {
				reason = *image.StateReason.Message
			}
			return "", fmt.Errorf("new image transitioned to %v: %v",
				state, reason)
		default:
			// RSI: log serious
			log.Printf("Unknown image state %v", state)
		}
	}

	// timed out

	return "", errors.New("Image was not done creating in 5 minutes")
}
