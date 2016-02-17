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

func (k *Kube) DeleteObject(o runtime.Object) {
	info, err := k.M.InfoForObject(o)
	if err != nil {
		// RSI: log cleanup error.
	}
	err = resource.NewHelper(info.Client, info.Mapping).Delete(info.Namespace, info.Name)
	if err != nil {
		// RSI: log cleanup error.
	}
	log.Printf("deleted %s.", info.Name)
}

func (k *Kube) CreateFromTemplate(
	template string, args ...string) ([]runtime.Object, error) {
	// RSI: make this configurable
	path := "/home/mlucy/go/src/github.com/rethinkdb/fusion-ops/templates/" + template
	body, err := exec.Command(path, args...).Output()
	if err != nil {
		return nil, err
	}
	var ret []runtime.Object
	defer func() {
		for i := range ret {
			go k.DeleteObject(ret[i])
		}
	}()
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(body), 4096)
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
		ret = append(ret, obj)
	}
	return ret, nil
}

type RDB struct {
	VolumeId string
	RC       runtime.Object
	SVC      runtime.Object
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
	return &RDB{
		VolumeId: volume,
		RC:       objs[0],
		SVC:      objs[1],
	}, nil
}

type Project struct {
	RDB *RDB
}

type ProjectOrError struct {
	P   *Project
	Err error
}

func (k *Kube) DeleteRDB(rdb *RDB) error {
	err := k.A.DeleteVolume(rdb.VolumeId)
	if err != nil {
		return err
	}
	fmt.Printf("deleted rdb")
	// RSI: todo
	return nil
}

func (k *Kube) CreateProject(name string) (*Project, error) {
	reschan := make(chan ProjectOrError)

	rdbvol := make(chan *aws.Volume)

	go func() {
		defer close(rdbvol)
		vol, err := k.A.CreateVolume(32)
		if err != nil {
			reschan <- ProjectOrError{Err: err}
			return
		}
		log.Printf("created volume %s", vol.Id)
		rdbvol <- vol
	}()

	go func() {
		vol := <-rdbvol
		if vol == nil {
			return
		}
		rdb, err := k.CreateRDB(name, vol.Id)
		if err != nil {
			err2 := k.A.DeleteVolume(vol.Id)
			if err2 != nil {
				// RSI: log cleanup failure.
			}
			reschan <- ProjectOrError{Err: err}
			return
		}
		log.Printf("created rdb %s", name)
		reschan <- ProjectOrError{P: &Project{RDB: rdb}}
	}()

	res := <-reschan
	if res.P == nil && res.Err == nil || res.P != nil && res.Err != nil {
		return nil, fmt.Errorf("unexpected ProjectOrError: %v", res)
	}
	return res.P, res.Err
}