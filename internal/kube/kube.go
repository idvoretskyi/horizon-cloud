package kube

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/rethinkdb/fusion-ops/internal/aws"

	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/yaml"
)

type Kube struct {
	M *resource.Mapper
	A *aws.AWS
}

type RDB struct {
	VolumeID string
	RC       runtime.Object
	SVC      runtime.Object
}

type Fusion struct {
	RC  runtime.Object
	SVC runtime.Object
}

type Frontend struct {
	VolumeID string
	RC       runtime.Object
	NGINXSVC runtime.Object
	SSHSVC   runtime.Object
}

type Project struct {
	RDB      *RDB
	Fusion   *Fusion
	Frontend *Frontend
}

func New(cluster string) *Kube {
	// RSI: should we be passing in a client config here?
	factory := util.NewFactory(nil)
	mapper, typer := factory.Object()
	return &Kube{
		M: &resource.Mapper{
			typer,
			mapper,
			resource.ClientMapperFunc(factory.ClientForMapping),
			factory.Decoder(true)},
		A: aws.New(cluster),
	}
}

func (k *Kube) DeleteObject(o runtime.Object) error {
	info, err := k.M.InfoForObject(o)
	if err != nil {
		return err
	}
	err = resource.NewHelper(info.Client, info.Mapping).Delete(info.Namespace, info.Name)
	if err != nil {
		return err
	}
	log.Printf("deleted %s.", info.Name)
	return nil
}

func (k *Kube) CreateFromTemplate(
	template string, args ...string) ([]runtime.Object, error) {

	// RSI: make this configurable
	path := "/home/mlucy/go/src/github.com/rethinkdb/fusion-ops/templates/" + template
	cmd := exec.Command(path, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	defer func() {
		// RSI: log cleanup failure
		cmd.Process.Kill()
		cmd.Wait()
	}()

	var objs []runtime.Object
	defer func() {
		for i := range objs {
			go func() {
				err := k.DeleteObject(objs[i])
				if err != nil {
					// RSI: log cleanup failure
				}
			}()
		}
	}()
	d := yaml.NewYAMLOrJSONDecoder(cmdReader, 4096)
	for {
		var ext runtime.RawExtension
		err = d.Decode(&ext)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		ext.RawJSON = bytes.TrimSpace(ext.RawJSON)
		// RSI: do validation?
		info, err := k.M.InfoForData(ext.RawJSON, path)
		if err != nil {
			return nil, err
		}
		info.Namespace = "default"
		obj, err := resource.NewHelper(info.Client, info.Mapping).
			Create(info.Namespace, true, info.Object)
		if err != nil {
			return nil, err
		}
		log.Printf("created %s.", info.Name)
		objs = append(objs, obj)
	}
	ret := objs
	objs = nil

	return ret, nil
}

func (k *Kube) CreateRDB(project string, volume string) (*RDB, error) {
	objs, err := k.CreateFromTemplate("rethinkdb.sh", project, volume)
	if err != nil {
		return nil, err
	}
	if len(objs) != 2 {
		// RSI: logging?
		return nil, fmt.Errorf("Internal error: template returned %d objects.", len(objs))
	}
	log.Printf("created rdb\n")
	// RSI: maybe do some asserts here that we actually have a
	// replication controller and service?
	return &RDB{
		VolumeID: volume,
		RC:       objs[0],
		SVC:      objs[1],
	}, nil
}

func (k *Kube) CreateFusion(project string) (*Fusion, error) {
	objs, err := k.CreateFromTemplate("fusion.sh", project)
	if err != nil {
		return nil, err
	}
	if len(objs) != 2 {
		// RSI: logging?
		return nil, fmt.Errorf("Internal error: template returned %d objects.", len(objs))
	}
	log.Printf("created fusion\n")
	// RSI: maybe do some asserts here that we actually have a
	// replication controller and service?
	return &Fusion{
		RC:  objs[0],
		SVC: objs[1],
	}, nil
}

func (k *Kube) CreateFrontend(project string, volume string) (*Frontend, error) {
	objs, err := k.CreateFromTemplate("frontend.sh", project, volume)
	if err != nil {
		return nil, err
	}
	if len(objs) != 3 {
		// RSI: logging?
		return nil, fmt.Errorf("Internal error: template returned %d objects.", len(objs))
	}
	log.Printf("created frontend\n")
	// RSI: maybe do some asserts here that we actually have a
	// replication controller and service?
	return &Frontend{
		VolumeID: volume,
		RC:       objs[0],
		NGINXSVC: objs[1],
		SSHSVC:   objs[2],
	}, nil
}

func (k *Kube) DeleteRDB(rdb *RDB) error {
	var errs []error
	errs = append(errs, k.A.DeleteVolume(rdb.VolumeID))
	errs = append(errs, k.DeleteObject(rdb.RC))
	errs = append(errs, k.DeleteObject(rdb.SVC))
	err := compositeErr(errs...)
	if err != nil {
		return err
	}
	log.Printf("deleted rdb")
	return nil
}

func (k *Kube) DeleteFusion(fusion *Fusion) error {
	var errs []error
	errs = append(errs, k.DeleteObject(fusion.RC))
	errs = append(errs, k.DeleteObject(fusion.SVC))
	err := compositeErr(errs...)
	if err != nil {
		return err
	}
	log.Printf("deleted fusion")
	return nil
}

func (k *Kube) DeleteFrontend(f *Frontend) error {
	var errs []error
	errs = append(errs, k.A.DeleteVolume(f.VolumeID))
	errs = append(errs, k.DeleteObject(f.RC))
	errs = append(errs, k.DeleteObject(f.NGINXSVC))
	errs = append(errs, k.DeleteObject(f.SSHSVC))
	err := compositeErr(errs...)
	if err != nil {
		return err
	}
	log.Printf("deleted frontend")
	return nil
}

type MaybeVolume struct {
	Volume *aws.Volume
	Err    error
}

type MaybeRDB struct {
	RDB *RDB
	Err error
}

type MaybeFusion struct {
	Fusion *Fusion
	Err    error
}

type MaybeFrontend struct {
	Frontend *Frontend
	Err      error
}

func (k *Kube) createWithVol(
	size int,
	volType string,
	callback func(MaybeVolume) error) {

	vol, err := k.A.CreateVolume(32, volType)
	if err != nil {
		err = callback(MaybeVolume{nil, err})
		if err != nil {
			// RSI: log cleanup failure
		}
		return
	}
	err = callback(MaybeVolume{vol, nil})
	if err != nil {
		err2 := k.A.DeleteVolume(vol.ID)
		if err2 != nil {
			// RSI: log cleanup failure.
		}
	}
}

func (k *Kube) CreateProject(name string) (*Project, error) {
	// RSI: don't hardcode volume sizes.

	rdbCh := make(chan MaybeRDB)
	fusionCh := make(chan MaybeFusion)
	frontendCh := make(chan MaybeFrontend)

	go k.createWithVol(32, aws.GP2, func(mv MaybeVolume) error {
		if mv.Err != nil {
			rdbCh <- MaybeRDB{nil, mv.Err}
			return nil
		}
		rdb, err := k.CreateRDB(name, mv.Volume.ID)
		rdbCh <- MaybeRDB{rdb, err}
		return err
	})

	go func() error {
		fusion, err := k.CreateFusion(name)
		fusionCh <- MaybeFusion{fusion, err}
		return err
	}()

	go k.createWithVol(4, aws.GP2, func(mv MaybeVolume) error {
		if mv.Err != nil {
			frontendCh <- MaybeFrontend{nil, mv.Err}
			return nil
		}
		frontend, err := k.CreateFrontend(name, mv.Volume.ID)
		frontendCh <- MaybeFrontend{frontend, err}
		return err
	})

	rdb := <-rdbCh
	fusion := <-fusionCh
	frontend := <-frontendCh

	err := compositeErr(rdb.Err, fusion.Err, frontend.Err)
	if err != nil {
		if rdb.RDB != nil {
			err := k.DeleteRDB(rdb.RDB)
			if err != nil {
				// RSI: log cleanup failure
			}
		}
		if fusion.Fusion != nil {
			err := k.DeleteFusion(fusion.Fusion)
			if err != nil {
				// RSI: log cleanup failure
			}
		}
		if frontend.Frontend != nil {
			err := k.DeleteFrontend(frontend.Frontend)
			if err != nil {
				// RSI: log cleanup failure
			}
		}
		return nil, err
	}

	return &Project{
		RDB:      rdb.RDB,
		Fusion:   fusion.Fusion,
		Frontend: frontend.Frontend,
	}, nil
}
