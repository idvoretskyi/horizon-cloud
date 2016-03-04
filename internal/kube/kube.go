package kube

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/aws"

	kapi "k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/yaml"
)

type Kube struct {
	C *client.Client
	M *resource.Mapper
	A *aws.AWS
}

type RDB struct {
	VolumeID string
	RC       *kapi.ReplicationController
	SVC      *kapi.Service
}

type Fusion struct {
	RC  *kapi.ReplicationController
	SVC *kapi.Service
}

type Frontend struct {
	VolumeID string
	RC       *kapi.ReplicationController
	NGINXSVC *kapi.Service
	SSHSVC   *kapi.Service
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
	client, err := factory.Client()
	if err != nil {
		// RSI: stop doing this when we support user clusters.
		log.Fatalf("unable to connect to Kube: %s", err)
	}
	return &Kube{
		C: client,
		M: &resource.Mapper{
			typer,
			mapper,
			resource.ClientMapperFunc(factory.ClientForMapping),
			factory.Decoder(true)},
		A: aws.New(cluster),
	}
}

func (k *Kube) Ready(p *Project) (bool, error) {
	for _, rc := range []*kapi.ReplicationController{
		p.RDB.RC, p.Fusion.RC, p.Frontend.RC} {
		log.Printf("checking readiness of RC %s", rc.Name)
		podlist, err := k.C.Pods(kapi.NamespaceDefault).List(kapi.ListOptions{
			LabelSelector: labels.SelectorFromSet(rc.Spec.Selector)})
		if err != nil {
			return false, err
		}
		// RSI: should we be asserting `len(podlist.Items)` is what we expect?
		for _, pod := range podlist.Items {
			log.Printf("checking status for PO %s", pod.Name)
			switch pod.Status.Phase {
			case kapi.PodPending:
				return false, nil
			case kapi.PodRunning:
			case kapi.PodSucceeded:
				return false, fmt.Errorf("pod exited unexpectedly")
			case kapi.PodFailed:
				return false, fmt.Errorf("pod failed unexpectedly")
			case kapi.PodUnknown:
				return false, fmt.Errorf("pod state unknown")
			default:
				return false, fmt.Errorf("unrecognized pod phase '%s'", pod.Status.Phase)
			}
			for _, condition := range pod.Status.Conditions {
				if condition.Type == kapi.PodReady {
					switch condition.Status {
					case kapi.ConditionTrue:
					case kapi.ConditionFalse:
						return false, nil
					case kapi.ConditionUnknown:
						return false, nil
					default:
						return false, fmt.Errorf("unrecognized status '%s'", condition.Status)
					}
				} else {
					// RSI: log unexpected condition
				}
			}
		}
	}
	return true, nil
}

func (k *Kube) Wait(p *Project) error {
	timeoutMin := time.Duration(5)
	backoff_ms := time.Duration(1000)

	timeout := time.NewTimer(timeoutMin * time.Minute)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timed out after %d minutes", timeoutMin)
		case <-time.After(backoff_ms * time.Millisecond):
			ready, err := k.Ready(p)
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
		backoff_ms = time.Duration(float64(backoff_ms) * 1.5)
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
		info.Namespace = kapi.NamespaceDefault
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

	rc, ok := objs[0].(*kapi.ReplicationController)
	if !ok {
		return nil, fmt.Errorf("unable to create RDB replication controller")
	}
	svc, ok := objs[1].(*kapi.Service)
	if !ok {
		return nil, fmt.Errorf("unable to create RDB service")
	}
	return &RDB{volume, rc, svc}, nil
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
	rc, ok := objs[0].(*kapi.ReplicationController)
	if !ok {
		return nil, fmt.Errorf("unable to create Fusion replication controller")
	}
	svc, ok := objs[1].(*kapi.Service)
	if !ok {
		return nil, fmt.Errorf("unable to create Fusion service")
	}
	return &Fusion{rc, svc}, nil
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
	rc, ok := objs[0].(*kapi.ReplicationController)
	if !ok {
		return nil, fmt.Errorf("unable to create Frontend replication controller")
	}
	nginxSVC, ok := objs[1].(*kapi.Service)
	if !ok {
		return nil, fmt.Errorf("unable to create NGINX service")
	}
	sshSVC, ok := objs[2].(*kapi.Service)
	if !ok {
		return nil, fmt.Errorf("unable to create SSH service")
	}
	return &Frontend{volume, rc, nginxSVC, sshSVC}, nil
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

func (k *Kube) createWithVol(
	size int,
	volType string,
	callback func(vol *aws.Volume, err error) error) {

	vol, err := k.A.CreateVolume(32, volType)
	if err != nil {
		err = callback(nil, err)
		if err != nil {
			// RSI: log cleanup failure
		}
		return
	}
	err = callback(vol, nil)
	if err != nil {
		err2 := k.A.DeleteVolume(vol.ID)
		if err2 != nil {
			// RSI: log cleanup failure.
		}
	}
}

func (k *Kube) CreateProject(conf api.Config) (*Project, error) {
	// RSI: Use `NumRDB`, `NumFusion`, and `NumFrontend`.

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

	rdbCh := make(chan MaybeRDB)
	fusionCh := make(chan MaybeFusion)
	frontendCh := make(chan MaybeFrontend)

	go k.createWithVol(conf.SizeRDB, aws.GP2, func(vol *aws.Volume, err error) error {
		if err != nil {
			rdbCh <- MaybeRDB{nil, err}
			return nil
		}
		rdb, err := k.CreateRDB(conf.Name, vol.ID)
		rdbCh <- MaybeRDB{rdb, err}
		return err
	})

	go func() {
		fusion, err := k.CreateFusion(conf.Name)
		fusionCh <- MaybeFusion{fusion, err}
	}()

	go k.createWithVol(conf.SizeFrontend, aws.GP2, func(vol *aws.Volume, err error) error {
		if err != nil {
			frontendCh <- MaybeFrontend{nil, err}
			return nil
		}
		frontend, err := k.CreateFrontend(conf.Name, vol.ID)
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
