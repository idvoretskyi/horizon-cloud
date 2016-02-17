package kube

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/yaml"
)

type Kube struct {
	m *resource.Mapper
}

func New() *Kube {
	// RSI: should we be passing in a client config here?
	factory := util.NewFactory(nil)
	mapper, typer := factory.Object()
	return &Kube{&resource.Mapper{
		typer,
		mapper,
		resource.ClientMapperFunc(factory.ClientForMapping),
		factory.Decoder(true),
	}}
}

func (k *Kube) CreateFromTemplate(
	template string, args ...string) ([]*runtime.Object, error) {
	// RSI: make this configurable
	path := "/home/mlucy/go/src/github.com/rethinkdb/fusion-ops/templates/" + template
	body, err := exec.Command(path, args...).Output()
	if err != nil {
		return nil, err
	}
	var ret []*runtime.Object
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
		info, err := k.m.InfoForData(ext.RawJSON, path)
		if err != nil {
			return nil, err
		}
		info.Namespace = "default"
		obj, err := resource.NewHelper(info.Client, info.Mapping).
			Create(info.Namespace, true, info.Object)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &obj)
	}
	return ret, nil
}

type RDB struct {
	rc  *runtime.Object
	svc *runtime.Object
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
		rc:  objs[0],
		svc: objs[1],
	}, nil
}
