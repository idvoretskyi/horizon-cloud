package gcloud

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pborman/uuid"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/compute/v1"
	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
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
	client       *http.Client
	compute      *compute.Service
	storage      *storage.Client
	storageAdmin *storage.AdminClient
	project      string
	zone         string
}

func New(serviceAccount *jwt.Config, project string, zone string) (*GCloud, error) {
	// TODO: make sure project and zone exist
	client := serviceAccount.Client(oauth2.NoContext)

	computeSv, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authOption := cloud.WithTokenSource(serviceAccount.TokenSource(ctx))

	storageClient, err := storage.NewClient(ctx, authOption)
	if err != nil {
		return nil, err
	}

	storageAdminClient, err := storage.NewAdminClient(ctx, project, authOption)
	if err != nil {
		return nil, err
	}

	return &GCloud{client, computeSv, storageClient, storageAdminClient,
		project, zone}, nil
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

func (g *GCloud) StorageClient() *storage.Client {
	return g.storage
}

func (g *GCloud) StorageAdminClient() *storage.AdminClient {
	return g.storageAdmin
}
