package aws

import (
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
