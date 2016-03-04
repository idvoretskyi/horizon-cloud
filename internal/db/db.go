package db

import (
	"errors"
	"fmt"
	"log"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/rethinkdb/fusion-ops/internal/api"
)

type DB struct {
	session *r.Session
}

var (
	ErrCanceled = errors.New("canceled")

	configs = r.DB("test").Table("configs")
)

func New() (*DB, error) {
	session, err := r.Connect(r.ConnectOpts{
		Address:           "localhost:28015",
		Database:          "test",
		MaxIdle:           10,
		MaxOpen:           10,
		HostDecayDuration: time.Second * 10,
	})
	if err != nil {
		return nil, err
	}
	return &DB{session}, nil
}

func setBasicError(typeName string, objectName string) error {
	return fmt.Errorf("internal error: unable to set %v `%v`", typeName, objectName)
}

func getBasicError(typeName string, name string) error {
	return fmt.Errorf("internal error: unable to get %v `%s`", typeName, name)
}

func (d *DB) SetConfig(c *Config) error {
	return d.setBasicType(configs, "config", c.Name, c)
}

func (d *DB) GetProjects(publicKey string) ([]api.Project, error) {
	cursor, err := configs.GetAllByIndex("PublicSSHKeys", publicKey).Run(d.session)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	var projects []api.Project
	var c Config
	for cursor.Next(&c) {
		projects = append(projects, api.Project{c.Name, "frontend-ssh-" + c.Name + ":22"})
	}
	return projects, nil
}

func (d *DB) GetConfig(name string) (*Config, error) {
	var c Config
	err := d.getBasicType(configs, "config", name, &c)
	return &c, err
}

func (d *DB) EnsureConfigConnectable(
	name string, allowClusterStart api.ClusterStartBool, keys []string) (*Config, error) {
	var out struct {
		Changes []struct {
			NewVal *Config `gorethink:"new_val"`
		} `gorethink:"changes"`
		FirstError string `gorethink:"first_error"`
	}

	var defaultConfig r.Term
	if allowClusterStart {
		defaultConfig = r.Expr(&Config{
			Config: api.Config{
				Name:         name,
				NumRDB:       1,
				SizeRDB:      10,
				NumFusion:    1,
				NumFrontend:  1,
				SizeFrontend: 1,
			},
		})
	} else {
		defaultConfig = r.Error("No such cluster.")
	}

	query := r.UUID().Do(func(uuid r.Term) interface{} {
		return configs.Get(name).Replace(func(row r.Term) interface{} {
			return row.Default(defaultConfig).Do(func(x r.Term) interface{} {
				return x.Merge(map[string]interface{}{
					"Version": uuid,
					"PublicSSHKeys": x.AtIndex("PublicSSHKeys").Default(
						[]string{}).SetUnion(keys),
				})
			})
		}, r.ReplaceOpts{ReturnChanges: "always"})
	})

	err := runOne(query, d.session, &out)
	if err != nil {
		log.Printf("Couldn't run EnsureConfigConnectable query: %s", err)
		return nil, getBasicError("ensureConfigConnectable", name)
	}

	if len(out.Changes) != 1 {
		log.Printf("Unexpected EnsureConfigConnectable response: %#v", out)
		return nil, getBasicError("ensureConfigConnectable", name)
	}

	if out.Changes[0].NewVal == nil {
		if allowClusterStart {
			log.Printf("Unexpected EnsureConfigConnectable response: %#v", out)
			return nil, getBasicError("ensureConfigConnectable", name)
		} else {
			return nil, fmt.Errorf("no cluster `%s`, do you want to deploy?", name)
		}
	}

	return out.Changes[0].NewVal, nil
}

type ConfigChange struct {
	OldVal *Config `gorethink:"old_val"`
	NewVal *Config `gorethink:"new_val"`
}

func (d *DB) configChangesLoop(out chan<- ConfigChange) {
	query := configs.Changes(r.ChangesOpts{IncludeInitial: true})
	for {
		ch := make(chan ConfigChange)
		cur, err := query.Run(d.session)
		if err != nil {
			// RSI: serious log
			log.Printf("SERIOUS FSCKING PROBLEM: %s", err)
			time.Sleep(time.Second * 5)
			continue
		}
		cur.Listen(out)
		for el := range ch {
			// RSI: sanity checks?
			out <- el
		}
		err = cur.Err()
		log.Printf("Channel closed, retrying: %s", err)
	}
}

func (d *DB) ConfigChanges(out chan<- ConfigChange) {
	go d.configChangesLoop(out)
}

func (d *DB) WaitConfigApplied(
	name string, version string, cancel <-chan struct{}) (*Config, error) {

	query := configs.Get(name).Changes(r.ChangesOpts{IncludeInitial: true})
	cur, err := query.Run(d.session)
	if err != nil {
		// RSI log
		return nil, getBasicError("config", name)
	}
	defer cur.Close()

	rows := make(chan ConfigChange)
	cur.Listen(rows)

	canceled := false
	for {
		select {
		case row, ok := <-rows:
			if !ok {
				if !canceled {
					// early changefeed close!
					// RSI: log cur.Err
					return nil, errors.New("internal error: changefeed closed unexpectedly")
				}
				return nil, ErrCanceled
			}

			if row.NewVal == nil {
				return nil, fmt.Errorf("configuration deleted")
			} else {
				if version != "" && row.NewVal.Version != version {
					return nil, fmt.Errorf("configuration superseded by %s", row.NewVal.Version)
				}
				if row.NewVal.AppliedVersion == row.NewVal.Version {
					return row.NewVal, nil
				}
			}

		case <-cancel:
			cur.Close()
			canceled = true
			cancel = nil
		}
	}
}

func runOne(query r.Term, session *r.Session, out interface{}) error {
	cur, err := query.Run(session)
	if err != nil {
		return err
	}
	return cur.One(out)
}

func (d *DB) getBasicType(table r.Term, typeName string, id string, i interface{}) error {
	err := runOne(table.Get(id), d.session, i)
	if err == r.ErrEmptyResult {
		return fmt.Errorf("%s `%s` does not exist", typeName, id)
	} else if err != nil {
		return getBasicError(typeName, id)
	}
	return nil
}

func (d *DB) setBasicType(table r.Term, typeName string, id string, i interface{}) error {
	res, err := table.Insert(i, r.InsertOpts{Conflict: "update"}).RunWrite(d.session)
	if err != nil {
		// RSI: log
		return setBasicError(typeName, id)
	}
	if res.Inserted+res.Unchanged+res.Replaced != 1 {
		// RSI: serious log
		return setBasicError(typeName, id)
	}
	return nil
}
