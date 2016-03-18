package db

import (
	"errors"
	"fmt"
	"log"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

type DB struct {
	session *r.Session
}

var (
	ErrCanceled = errors.New("canceled")

	configs = r.DB("hzc_api").Table("configs")
	users   = r.DB("hzc_api").Table("users")
	domains = r.DB("hzc_api").Table("domains")
)

func New() (*DB, error) {
	session, err := r.Connect(r.ConnectOpts{
		Address:           "localhost:28015",
		Database:          "test",
		AuthKey:           "hzc",
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

func (d *DB) SetConfig(c api.Config) (*api.Config, error) {
	q := configs.Insert(c, r.InsertOpts{Conflict: "update", ReturnChanges: "always"})

	var resp struct {
		Errors     int    `gorethink:"errors"`
		FirstError string `gorethink:"first_error"`
		Changes    []struct {
			NewVal *api.Config `gorethink:"new_val"`
		}
	}
	err := runOne(q, d.session, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errors != 0 {
		return nil, errors.New(resp.FirstError)
	}

	if len(resp.Changes) != 1 {
		return nil, errors.New("Unexpected number of changes in response")
	}

	return resp.Changes[0].NewVal, nil
}

func (d *DB) GetProjectsByKey(publicKey string) ([]api.Project, error) {
	cursor, err := configs.GetAllByIndex("Users",
		r.Args(users.GetAllByIndex("PublicSSHKeys", publicKey).
			Field("id").CoerceTo("array"))).Run(d.session)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	var projects []api.Project
	var c api.Config
	for cursor.Next(&c) {
		projects = append(projects, api.ProjectFromName(c.Name))
	}
	return projects, nil
}

func (d *DB) GetByDomain(domainName string) (*api.Project, error) {
	var domain api.Domain
	err := runOne(domains.Get(domainName), d.session, &domain)
	if err != nil {
		if err != r.ErrEmptyResult {
			return nil, err
		}
		return nil, nil
	}
	project := api.ProjectFromName(domain.Project)
	return &project, nil
}

func (d *DB) GetConfig(name string) (*api.Config, error) {
	var c api.Config
	err := d.getBasicType(configs, "config", util.TrueName(name), &c)
	return &c, err
}

func (d *DB) UserCreate(name string) error {
	q := users.Insert(api.User{Name: name, PublicSSHKeys: []string{}})
	_, err := q.RunWrite(d.session)
	return err
}

func (d *DB) UserGet(name string) (*api.User, error) {
	var user api.User
	err := d.getBasicType(users, "user", name, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *DB) UserAddKeys(name string, keys []string) error {
	q := users.Get(name).Update(func(row r.Term) r.Term {
		return r.Expr(map[string]interface{}{
			"PublicSSHKeys": row.Field("PublicSSHKeys").SetUnion(keys)})
	})
	resp, err := q.RunWrite(d.session)
	if err != nil {
		return err
	}
	if resp.Skipped > 0 {
		return fmt.Errorf("user `%s` does not exist", name)
	}
	return nil
}

func (d *DB) UserDelKeys(name string, keys []string) error {
	q := users.Get(name).Update(func(row r.Term) r.Term {
		return r.Expr(map[string]interface{}{
			"PublicSSHKeys": row.Field("PublicSSHKeys").SetDifference(keys)})
	})
	resp, err := q.RunWrite(d.session)
	if err != nil {
		return err
	}
	if resp.Skipped > 0 {
		return fmt.Errorf("user `%s` does not exist", name)
	}
	return nil
}

func (d *DB) SetDomain(domain api.Domain) error {
	return d.setBasicType(domains, "domain", domain.Domain, &domain)
}

func (d *DB) GetDomainsByProject(project string) ([]string, error) {
	cursor, err := domains.GetAllByIndex("Project", project).Run(d.session)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	domains := []string{}
	var dom api.Domain
	for cursor.Next(&dom) {
		domains = append(domains, dom.Domain)
	}
	return domains, nil
}

func (d *DB) EnsureConfigConnectable(
	name string, allowClusterStart api.ClusterStartBool) (*api.Config, error) {

	if allowClusterStart {
		var out struct {
			Changes []struct {
				NewVal *api.Config `gorethink:"new_val"`
			} `gorethink:"changes"`
			FirstError string `gorethink:"first_error"`
		}

		defaultConfig := r.Expr(api.ConfigFromDesired(api.DefaultDesiredConfig(name)))
		query := configs.Insert(defaultConfig, r.InsertOpts{ReturnChanges: "always"})
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
			log.Printf("Unexpected EnsureConfigConnectable response: %#v", out)
			return nil, getBasicError("ensureConfigConnectable", name)
		}
		return out.Changes[0].NewVal, nil

	} else {
		var c api.Config
		query := configs.Get(util.TrueName(name))
		err := runOne(query, d.session, &c)
		if err != nil {
			log.Printf("EnsureConfigConnectable for `%s` failed: %s", name, err)
			return nil, fmt.Errorf("no cluster `%s`", name)
		}
		return &c, err
	}
}

type ConfigChange struct {
	OldVal *api.Config `gorethink:"old_val"`
	NewVal *api.Config `gorethink:"new_val"`
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
	name string, version string, cancel <-chan struct{}) (*api.Config, error) {

	query := configs.Get(util.TrueName(name)).Changes(r.ChangesOpts{IncludeInitial: true})
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

func (d *DB) getBasicType(
	table r.Term, typeName string, id string, i interface{}) error {

	err := runOne(table.Get(id), d.session, i)
	if err == r.ErrEmptyResult {
		return fmt.Errorf("%s `%s` does not exist", typeName, id)
	} else if err != nil {
		return getBasicError(typeName, id)
	}
	return nil
}

func (d *DB) setBasicType(
	table r.Term, typeName string, id string, i interface{}) error {

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
