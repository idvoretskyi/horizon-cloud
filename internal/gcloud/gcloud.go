package gcloud

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pborman/uuid"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

type DiskType string

var (
	DiskTypeStandard DiskType = "pd-standard"
	DiskTypeSSD      DiskType = "pd-ssd"
)

type Disk struct {
	Name   string
	SizeGB int64
}

type GCloud struct {
	client  *http.Client
	compute *compute.Service
	project string
	zone    string
}

func New(project string, zone string) (*GCloud, error) {
	ctx := context.TODO()

	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		return nil, err
	}

	computeSv, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	// TODO: make sure project and zone exist

	return &GCloud{client, computeSv, project, zone}, nil
}

func (g *GCloud) CreateDisk(sizeGB int64, disktype DiskType) (*Disk, error) {
	name := "uuid-" + uuid.New()

	diskTypeURL := "https://www.googleapis.com/compute/v1/projects/" +
		g.project + "/zones/" + g.zone + "/diskTypes/" + string(disktype)

	log.Printf("Creating disk %v", name)

	_, err := g.compute.Disks.Insert(g.project, g.zone, &compute.Disk{
		Name:   name,
		SizeGb: sizeGB,
		Type:   diskTypeURL,
	}).Do()
	if err != nil {
		return nil, err
	}

	for {
		time.Sleep(time.Second)
		disk, err := g.compute.Disks.Get(g.project, g.zone, name).Do()
		if err != nil {
			return nil, err
		}

		switch disk.Status {
		case "CREATING":
			// do nothing, repeat
		case "FAILED":
			return nil, fmt.Errorf("disk failed to create")
		case "READY":
			return &Disk{
				Name:   disk.Name,
				SizeGB: disk.SizeGb,
			}, nil
		default:
			return nil, fmt.Errorf("disk entered unexpected status %#v", disk.Status)
		}
	}
}

func (g *GCloud) DeleteDisk(name string) error {
	log.Printf("Deleting disk %v", name)
	_, err := g.compute.Disks.Delete(g.project, g.zone, name).Do()
	return err
}
