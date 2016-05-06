package kube

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/rethinkdb/horizon-cloud/internal/gcloud"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"

	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	kcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	kutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/yaml"
)

const userNamespace = "user"

type Kube struct {
	TemplatePath string
	C            *client.Client
	Conf         *client.Config
	M            *resource.Mapper
	G            *gcloud.GCloud
}

type RDB struct {
	VolumeID string
	RC       *kapi.ReplicationController
	SVC      *kapi.Service
}

type Horizon struct {
	RC  *kapi.ReplicationController
	SVC *kapi.Service
}

type Project struct {
	RDB     *RDB
	Horizon *Horizon
}

var newMu sync.Mutex

func New(templatePath string, gc *gcloud.GCloud) *Kube {
	newMu.Lock() // kutil.NewFactory is racy.
	// RSI: should we be passing in a client config here?
	factory := kutil.NewFactory(nil)
	newMu.Unlock()
	mapper, typer := factory.Object()
	client, err := factory.Client()
	if err != nil {
		// RSI: stop doing this when we support user clusters.
		log.Fatalf("unable to connect to Kube: %s", err)
	}
	conf, err := factory.ClientConfig()
	if err != nil {
		log.Fatalf("unable to get client config: %s", err)
	}
	return &Kube{
		TemplatePath: templatePath,
		C:            client,
		Conf:         conf,
		M: &resource.Mapper{
			ObjectTyper:  typer,
			RESTMapper:   mapper,
			ClientMapper: resource.ClientMapperFunc(factory.ClientForMapping),
			Decoder:      factory.Decoder(true)},
		G: gc,
	}
}

func (k *Kube) GetHorizonPodsForProject(projectName string) ([]string, error) {
	trueName := util.TrueName(projectName)
	pods, err := k.C.Pods("user").List(kapi.ListOptions{
		LabelSelector: labels.Set(map[string]string{
			"app":     "horizon",
			"project": trueName,
		}).AsSelector(),
	})
	if err != nil {
		return nil, err
	}

	ret := make([]string, len(pods.Items))
	for index, pod := range pods.Items {
		ret[index] = pod.Name
	}
	return ret, nil
}

// Usually all you want to set are `PodName`, `Command`, and maybe `In`.
type ExecOptions kcmd.ExecOptions

func (k *Kube) Exec(eo ExecOptions) (string, string, error) {
	// RSI: Make these limited in size.
	var resBuf bytes.Buffer
	var errBuf bytes.Buffer

	if eo.Namespace == "" {
		eo.Namespace = "user"
	}
	if eo.PodName == "" {
		return "", "", fmt.Errorf("Kube.Exec requires a podname")
	}

	if eo.In != nil {
		eo.Stdin = true
	}

	// If the user sets these we just write to their writers and return
	// empty string.
	if eo.Out == nil {
		eo.Out = &resBuf
	}
	if eo.Err == nil {
		eo.Err = &errBuf
	}

	if eo.Executor == nil {
		eo.Executor = &kcmd.DefaultRemoteExecutor{}
	}
	if eo.Client == nil {
		eo.Client = k.C
	}
	if eo.Config == nil {
		eo.Config = k.Conf
	}

	real_eo := kcmd.ExecOptions(eo)
	err := real_eo.Validate()
	if err != nil {
		return "", "", err
	}
	err = real_eo.Run()
	// We return the buffers even if there was an error because there
	// might still be useful stuff in them.
	return resBuf.String(), errBuf.String(), err
}

func (k *Kube) Ready(p *Project) (bool, error) {
	for _, rc := range []*kapi.ReplicationController{p.RDB.RC, p.Horizon.RC} {
		log.Printf("checking readiness of RC %s", rc.Name)
		podlist, err := k.C.Pods(userNamespace).List(kapi.ListOptions{
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
	timeoutMin := 5 * time.Minute
	backoff_ms := 1000 * time.Millisecond
	backoff_ms_increment := 100 * time.Millisecond

	timeout := time.NewTimer(timeoutMin)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timed out after %v minutes", timeoutMin)
		case <-time.After(backoff_ms):
			log.Printf("Polling for readiness")
			ready, err := k.Ready(p)
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
		backoff_ms += backoff_ms_increment
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

	path := k.TemplatePath + template
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
		for _, o := range objs {
			o := o
			go func() {
				err := k.DeleteObject(o)
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
		info.Namespace = userNamespace
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

func (k *Kube) CreateHorizon(project string) (*Horizon, error) {
	objs, err := k.CreateFromTemplate("horizon.sh", project)
	if err != nil {
		return nil, err
	}
	if len(objs) != 2 {
		// RSI: logging?
		return nil, fmt.Errorf("Internal error: template returned %d objects.", len(objs))
	}
	log.Printf("created horizon\n")
	rc, ok := objs[0].(*kapi.ReplicationController)
	if !ok {
		return nil, fmt.Errorf("unable to create Horizon replication controller")
	}
	svc, ok := objs[1].(*kapi.Service)
	if !ok {
		return nil, fmt.Errorf("unable to create Horizon service")
	}
	return &Horizon{rc, svc}, nil
}

func (k *Kube) DeleteRDB(rdb *RDB) error {
	var errs []error
	errs = append(errs, k.G.DeleteDisk(rdb.VolumeID))
	errs = append(errs, k.DeleteObject(rdb.RC))
	errs = append(errs, k.DeleteObject(rdb.SVC))
	err := compositeErr(errs...)
	if err != nil {
		return err
	}
	log.Printf("deleted rdb")
	return nil
}

func (k *Kube) DeleteHorizon(horizon *Horizon) error {
	var errs []error
	errs = append(errs, k.DeleteObject(horizon.RC))
	errs = append(errs, k.DeleteObject(horizon.SVC))
	err := compositeErr(errs...)
	if err != nil {
		return err
	}
	log.Printf("deleted horizon")
	return nil
}

func (k *Kube) createWithVol(
	size int,
	volType gcloud.DiskType,
	callback func(vol *gcloud.Disk, err error) error) {

	vol, err := k.G.CreateDisk(int64(size), volType)
	if err != nil {
		err = callback(nil, err)
		if err != nil {
			// RSI: log cleanup failure
		}
		return
	}
	err = callback(vol, nil)
	if err != nil {
		err2 := k.G.DeleteDisk(vol.Name)
		if err2 != nil {
			// RSI: log cleanup failure.
		}
	}
}

func (k *Kube) getRC(name string) (*kapi.ReplicationController, error) {
	rc, err := k.C.ReplicationControllers(userNamespace).Get(name)
	if err != nil {
		if serr, ok := err.(*kerrors.StatusError); ok && serr.Status().Code == 404 {
			return nil, nil
		}
		return nil, err
	}
	return rc, nil
}

func (k *Kube) EnsureProject(
	trueName string, conf types.KubeConfig) (*Project, error) {
	// RSI: Use `NumRDB` and `NumHorizon`

	type MaybeRDB struct {
		RDB *RDB
		Err error
	}

	type MaybeHorizon struct {
		Horizon *Horizon
		Err     error
	}

	rdbCh := make(chan MaybeRDB)
	horizonCh := make(chan MaybeHorizon)

	go func() {
		rdbCh <- func() MaybeRDB {
			rc, err := k.getRC("r0-" + trueName)
			if err != nil {
				return MaybeRDB{nil, err}
			}

			// RSI: check that the replicationcontroller is what we want.
			if rc != nil {
				svc, err := k.C.Services(userNamespace).Get("r-" + trueName)
				if err != nil {
					// RSI: we should be creating replicationcontrollers and
					// services separately so we don't have to abort here.
					return MaybeRDB{nil, err}
				}
				volName := rc.Spec.Template.Spec.Volumes[0].GCEPersistentDisk.PDName
				log.Printf("%s already exists with volume %s", "r0-"+trueName, volName)
				return MaybeRDB{&RDB{volName, rc, svc}, nil}
			}

			var ret MaybeRDB
			k.createWithVol(conf.SizeRDB, gcloud.DiskTypeSSD,
				func(vol *gcloud.Disk, err error) error {
					if err != nil {
						ret = MaybeRDB{nil, err}
						return nil
					}
					rdb, err := k.CreateRDB(trueName, vol.Name)
					ret = MaybeRDB{rdb, err}
					return err
				})
			return ret
		}()
	}()

	go func() {
		horizonCh <- func() MaybeHorizon {
			rc, err := k.getRC("h0-" + trueName)
			if err != nil {
				return MaybeHorizon{nil, err}
			}
			if rc != nil {
				svc, err := k.C.Services(userNamespace).Get("h-" + trueName)
				if err != nil {
					return MaybeHorizon{nil, err}
				}
				log.Printf("%s already exists", "h0-"+trueName)
				return MaybeHorizon{&Horizon{rc, svc}, nil}
			}

			horizon, err := k.CreateHorizon(trueName)
			return MaybeHorizon{horizon, err}
		}()
	}()

	rdb := <-rdbCh
	horizon := <-horizonCh

	err := compositeErr(rdb.Err, horizon.Err)
	if err != nil {
		if rdb.RDB != nil {
			err := k.DeleteRDB(rdb.RDB)
			if err != nil {
				// RSI: log cleanup failure
			}
		}
		if horizon.Horizon != nil {
			err := k.DeleteHorizon(horizon.Horizon)
			if err != nil {
				// RSI: log cleanup failure
			}
		}
		return nil, err
	}

	return &Project{
		RDB:     rdb.RDB,
		Horizon: horizon.Horizon,
	}, nil
}
