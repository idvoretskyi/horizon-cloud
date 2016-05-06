package db

import (
	"errors"
	"fmt"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

type DBConnection struct {
	session *r.Session
}

var (
	ErrCanceled = errors.New("canceled")

	configs = r.DB("hzc_api").Table("configs")
	users   = r.DB("hzc_api").Table("users")
	domains = r.DB("hzc_api").Table("domains")
)

func New() (*DBConnection, error) {
	session, err := r.Connect(r.ConnectOpts{
		Address:           "localhost:28015",
		AuthKey:           "hzc",
		MaxIdle:           10,
		MaxOpen:           10,
		HostDecayDuration: time.Second * 10,
	})
	if err != nil {
		return nil, err
	}
	return &DBConnection{session}, nil
}

func (d *DBConnection) WithLogger(l *hzlog.Logger) *DB {
	return &DB{d, l}
}

type DB struct {
	*DBConnection
	log *hzlog.Logger
}

func setBasicError(typeName string, objectName string) error {
	return fmt.Errorf("internal error: unable to set %v `%v`", typeName, objectName)
}

func getBasicError(typeName string, name string) error {
	return fmt.Errorf("internal error: unable to get %v `%s`", typeName, name)
}

func (d *DB) SetConfig(c types.Config) (*types.Config, error) {
	q := configs.Insert(c, r.InsertOpts{Conflict: "update", ReturnChanges: "always"})

	var resp struct {
		Errors     int    `gorethink:"errors"`
		FirstError string `gorethink:"first_error"`
		Changes    []struct {
			NewVal *types.Config `gorethink:"new_val"`
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

func (d *DB) MaybeUpdateHorizonConfig(
	project string, hzConf types.HorizonConfig) (int64, error) {
	q := r.Expr(hzConf).Do(func(hzc r.Term) r.Term {
		return configs.Get(util.TrueName(project)).Update(func(config r.Term) r.Term {
			return r.Branch(
				config.Field("HorizonConfig").Eq(hzc).Default(false),
				nil,
				map[string]r.Term{
					"HorizonConfig":        r.Expr(hzc),
					"HorizonConfigVersion": config.Field("HorizonConfigVersion").Default(0).Add(1),
				})
		}, r.UpdateOpts{ReturnChanges: true})
	})
	res, err := q.RunWrite(d.session)
	if err != nil {
		return 0, err
	}

	if res.Unchanged == 1 {
		return 0, nil
	}
	if res.Replaced != 1 {
		return 0, fmt.Errorf("maybeUpdateHorizonConfig failed to update (%v)", res)
	}
	if len(res.Changes) != 1 {
		return 0, fmt.Errorf("maybeUpdateHorizonConfig got unexpected changes (%v)", res)
	}
	newVal := res.Changes[0].NewValue
	if newVal == nil {
		return 0, fmt.Errorf("maybeUpdateHorizonConfig got unexpected changes (%v)", res)
	}

	var newVer int64
	switch m := newVal.(type) {
	case map[string]interface{}:
		newVersion := m["HorizonConfigVersion"]
		switch v := newVersion.(type) {
		case float64:
			newVer = int64(v)
		default:
			return 0, fmt.Errorf("maybeUpdateHorizonConfig got unexpected changes (%v)", res)
		}
	default:
		return 0, fmt.Errorf("maybeUpdateHorizonConfig got unexpected changes (%v)", res)
	}

	return newVer, nil
}

func (d *DB) WaitForHorizonConfigVersion(project string, version int64) (string, error) {
	// RSI: do this
	return "", nil
}

func (d *DB) HorizonConfigHashMatches(project string, confHash string) (bool, error) {
	q := configs.Get(util.TrueName(project)).
		Field("HorizonConfigHash").
		Default("")
	var oldConfHash string
	err := runOne(q, d.session, &confHash)
	if err != nil {
		return false, err
	}
	return oldConfHash == confHash, nil
}

func (d *DB) SetHorizonConfigHash(project string, confHash string) error {
	q := configs.Get(util.TrueName(project)).Update(map[string]string{
		"HorizonConfigHash": confHash})
	res, err := q.RunWrite(d.session)
	if err != nil {
		return err
	}
	if res.Replaced == 0 && res.Unchanged == 0 {
		return fmt.Errorf("Unable to update Horizon config hash (%v).", res)
	}
	return nil
}

func (d *DB) GetUsersByKey(publicKey string) ([]string, error) {
	q := users.GetAllByIndex("PublicSSHKeys", publicKey)
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("Couldn't get users by key: %v", err)
		return nil, err
	}
	defer cursor.Close()
	var users []string
	var u types.User
	for cursor.Next(&u) {
		users = append(users, u.Name)
	}
	return users, nil
}

func (d *DB) GetProjectsByKey(publicKey string) ([]types.Project, error) {
	q := configs.GetAllByIndex("Users",
		r.Args(users.GetAllByIndex("PublicSSHKeys", publicKey).
			Field("id").CoerceTo("array")))
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("Couldn't get projects by key: %v", err)
		return nil, err
	}
	defer cursor.Close()
	var projects []types.Project
	var c types.Config
	for cursor.Next(&c) {
		projects = append(projects, types.ProjectFromName(c.Name))
	}
	return projects, nil
}

func (d *DB) GetProjectsByUsers(users []string) ([]types.Project, error) {
	q := configs.GetAllByIndex("Users", r.Args(users))
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("Couldn't get projects by key: %v", err)
		return nil, err
	}
	defer cursor.Close()
	var projects []types.Project
	var c types.Config
	for cursor.Next(&c) {
		projects = append(projects, types.ProjectFromName(c.Name))
	}
	return projects, nil
}

func (d *DB) GetByDomain(domainName string) (*types.Project, error) {
	var domain types.Domain
	err := runOne(domains.Get(domainName), d.session, &domain)
	if err != nil {
		if err != r.ErrEmptyResult {
			return nil, err
		}
		return nil, nil
	}
	project := types.ProjectFromName(domain.Project)
	return &project, nil
}

func (d *DB) GetConfig(name string) (*types.Config, error) {
	var c types.Config
	err := d.getBasicType(configs, "config", util.TrueName(name), &c)
	return &c, err
}

func (d *DB) UserCreate(name string) error {
	q := users.Insert(types.User{Name: name, PublicSSHKeys: []string{}})
	_, err := q.RunWrite(d.session)
	return err
}

func (d *DB) UserGet(name string) (*types.User, error) {
	var user types.User
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

func (d *DB) SetDomain(domain types.Domain) error {
	return d.setBasicType(domains, "domain", domain.Domain, &domain)
}

func (d *DB) GetDomainsByProject(project string) ([]string, error) {
	cursor, err := domains.GetAllByIndex("Project", project).Run(d.session)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	domains := []string{}
	var dom types.Domain
	for cursor.Next(&dom) {
		domains = append(domains, dom.Domain)
	}
	return domains, nil
}

type ConfigChange struct {
	OldVal *types.Config `gorethink:"old_val"`
	NewVal *types.Config `gorethink:"new_val"`
}

func (d *DB) configChangesLoop(out chan<- ConfigChange) {
	query := configs.Changes(r.ChangesOpts{IncludeInitial: true})
	for {
		ch := make(chan ConfigChange)
		cur, err := query.Run(d.session)
		if err != nil {
			d.log.Error("Couldn't query for config changes: %s", err)
			time.Sleep(time.Second * 5)
			continue
		}
		cur.Listen(out)
		for el := range ch {
			// RSI: sanity checks?
			out <- el
		}
		err = cur.Err()
		d.log.Error("Config changes loop ended: %v", err)
	}
}

func (d *DB) ConfigChanges(out chan<- ConfigChange) {
	go d.configChangesLoop(out)
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
		d.log.Error("Couldn't set %v %#v: %v", typeName, id, err)
		return setBasicError(typeName, id)
	}
	if res.Inserted+res.Unchanged+res.Replaced != 1 {
		d.log.Error("Couldn't set %v %#v: unexpected result counts", typeName, id)
		return setBasicError(typeName, id)
	}
	return nil
}
