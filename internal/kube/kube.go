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
	fmt.Printf("deleted rdb")
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
	fmt.Printf("deleted fusion")
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
	fmt.Printf("deleted frontend")
	return nil
}

func (k *Kube) createWithVol(
	size int,
	abort chan error,
	callback func(*aws.Volume) error) {

	vol, err := k.A.CreateVolume(32)
	if err != nil {
		abort <- err
		return
	}
	err = callback(vol)
	if err != nil {
		err2 := k.A.DeleteVolume(vol.Id)
		if err2 != nil {
			// RSI: log cleanup failure.
		}
		abort <- err
	}
}

func (k *Kube) CreateProject(name string) (*Project, error) {
	// RSI: don't hardcode volume sizes.

	// Make sure that no more than this many errors can be written to `abort`.
	MAX_ERRORS := 1024
	// We don't close `abort` because we want to return early if there's
	// an error and let the goroutines below continue writing to it.
	abort := make(chan error, MAX_ERRORS)

	rdbCh := make(chan *RDB)
	fusionCh := make(chan *Fusion)
	frontendCh := make(chan *Frontend)

	go k.createWithVol(32, abort, func(vol *aws.Volume) error {
		defer close(rdbCh)
		rdb, err := k.CreateRDB(name, vol.Id)
		if err != nil {
			return err
		}
		log.Printf("created rdb %s", name)
		rdbCh <- rdb
		return nil
	})

	go func() {
		defer close(fusionCh)
		fusion, err := k.CreateFusion(name)
		if err != nil {
			abort <- err
			return
		}
		log.Printf("created fusion %s", name)
		fusionCh <- fusion
	}()

	go k.createWithVol(4, abort, func(vol *aws.Volume) error {
		defer close(frontendCh)
		frontend, err := k.CreateFrontend(name, vol.Id)
		if err != nil {
			return err
		}
		log.Printf("created frontend %s", name)
		frontendCh <- frontend
		return nil
	})

	// We give this a buffer of size 1 so that if there's a logic error
	// that causes `abort` to be pulsed without causing one of the
	// components of the project to receive `nil`, we won't hang forever
	// on the write to `projCh`.
	projCh := make(chan *Project, 1)
	go func() {
		// We don't close `projCh` because we want the `select` below to
		// pick up the error in the error case.
		rdb := <-rdbCh
		fusion := <-fusionCh
		frontend := <-frontendCh
		// At least one of these is nil iff an error occurs.
		if rdb == nil || fusion == nil || frontend == nil {
			// This error should never be seen, but by putting it into abort
			// logic errors that break the above invariant won't cause hangs.
			abort <- fmt.Errorf("unexpected empty abort queue")
			if rdb != nil {
				err := k.DeleteRDB(rdb)
				if err != nil {
					// RSI: log cleanup failure
				}
			}
			if fusion != nil {
				err := k.DeleteFusion(fusion)
				if err != nil {
					// RSI: log cleanup failure
				}
			}
			if frontend != nil {
				err := k.DeleteFrontend(frontend)
				if err != nil {
					// RSI: log cleanup failure
				}
			}
			return
		}
		projCh <- &Project{
			RDB:      rdb,
			Fusion:   fusion,
			Frontend: frontend,
		}
	}()

	select {
	case err := <-abort:
		if err == nil {
			return nil, fmt.Errorf("unexpected empty error")
		}
		return nil, err
	case proj := <-projCh:
		if proj == nil {
			return nil, fmt.Errorf("unexpected empty project")
		}
		return proj, nil
	}
}
