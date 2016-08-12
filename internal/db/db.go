package db

import (
	"errors"
	"fmt"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/types"
)

type DBConnection struct {
	session *r.Session
}

var (
	ErrCanceled = errors.New("canceled")

	projects = r.DB("web_backend").Table("projects")
	domains  = r.DB("web_backend").Table("domains")
	users    = r.DB("web_backend_internal").Table("users")
)

type hzUser struct {
	Name string `gorethink:"id"`
	Data struct {
		PublicSSHKeys []string `gorethink:"keys"`
	} `gorethink:"data"`
}

func New(addr string) (*DBConnection, error) {
	session, err := r.Connect(r.ConnectOpts{
		Address:           addr,
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

type projectWriteResp struct {
	Errors     int    `gorethink:"errors"`
	Skipped    int    `gorethink:"skipped"`
	Unchanged  int    `gorethink:"unchanged"`
	FirstError string `gorethink:"first_error"`
	Changes    []struct {
		NewVal *types.Project `gorethink:"new_val"`
	} `gorethink:"changes"`
}

func (d *DB) runProjectWriteDetailed(
	q r.Term) (*types.Project, *projectWriteResp, error) {
	var resp projectWriteResp
	err := runOne(q, d.session, &resp)

	if err != nil {
		return nil, nil, err
	}
	if resp.Errors != 0 {
		return nil, nil, errors.New(resp.FirstError)
	}
	if resp.Skipped != 0 {
		return nil, nil, fmt.Errorf("project was deleted before write could complete")
	}
	if len(resp.Changes) != 1 {
		return nil, nil, fmt.Errorf("Unexpected number of changes in response: %v", resp)
	}
	ret := resp.Changes[0].NewVal
	if ret == nil {
		return nil, nil, fmt.Errorf("internal error: no value after write")
	}
	return ret, &resp, nil
}

func (d *DB) runProjectWrite(q r.Term) (*types.Project, error) {
	p, _, err := d.runProjectWriteDetailed(q)
	return p, err
}

func (d *DB) UpdateProject(projectPatch types.Project) (bool, error) {
	q := projects.Get(projectPatch.ID).Update(projectPatch)
	res, err := q.RunWrite(d.session)
	if err != nil {
		return false, err
	}
	return res.Replaced == 1, nil
}

func (d *DB) DeleteProject(projectID types.ProjectID) error {
	q := projects.Get(projectID).Delete()
	_, err := q.RunWrite(d.session)
	return err
}

func (d *DB) MaybeUpdateHorizonConfig(
	projectID types.ProjectID, hzConf types.HorizonConfig) (int64, string, error) {
	hcv := "HorizonConfigVersion"
	q := r.Expr(hzConf).Do(func(hzc r.Term) r.Term {
		return projects.Get(projectID).Update(func(config r.Term) r.Term {
			return r.Branch(
				config.Field("HorizonConfig").Eq(hzc).Default(false),
				nil,
				map[string]interface{}{
					"HorizonConfig": r.Expr(hzc),
					hcv: map[string]interface{}{
						"Desired": config.Field(hcv).Field("Desired").Default(0).Add(1),
					},
				})
		}, r.UpdateOpts{ReturnChanges: "always"})
	})
	project, resp, err := d.runProjectWriteDetailed(q)
	if err != nil {
		return 0, "", err
	}
	if resp.Unchanged != 0 {
		if project.HorizonConfigVersion.Desired == project.HorizonConfigVersion.Applied {
			// We return a special value if nothing changed so that we can skip waiting.
			return 0, "", nil
		}
		if project.HorizonConfigVersion.Desired == project.HorizonConfigVersion.Error {
			return 0, project.HorizonConfigVersion.LastError, nil
		}
	}
	return project.HorizonConfigVersion.Desired, "", nil
}

type HZStateType int

const (
	HZError      HZStateType = 0
	HZApplied    HZStateType = 1
	HZSuperseded HZStateType = 2
	HZDeleted    HZStateType = 3
)

type HZState struct {
	Typ       HZStateType
	LastError string
}

func (d *DB) WaitForHorizonConfigVersion(
	projectID types.ProjectID, version int64) (HZState, error) {
	q := projects.Get(projectID).Changes(r.ChangesOpts{IncludeInitial: true})
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("WaitForHorizonConfigVersion(%v, %d): %v", projectID, version, err)
		return HZState{}, err
	}
	defer cursor.Close()
	var c ProjectChange
	for cursor.Next(&c) {
		if c.NewVal == nil {
			return HZState{Typ: HZDeleted}, nil
		}
		if c.NewVal.HorizonConfigVersion.Applied == version {
			return HZState{Typ: HZApplied}, nil
		}
		if c.NewVal.HorizonConfigVersion.Error == version {
			return HZState{
				Typ:       HZError,
				LastError: c.NewVal.HorizonConfigVersion.LastError,
			}, nil
		}
		if c.NewVal.HorizonConfigVersion.Applied > version ||
			c.NewVal.HorizonConfigVersion.Error > version {
			return HZState{Typ: HZSuperseded}, nil
		}
	}

	err = cursor.Err()
	if err != nil {
		err = fmt.Errorf("Changefeed aborted unexpectedly.")
	}

	d.log.Error("WaitForHorizonConfigVersion(%v, %d): %v", projectID, version, err)
	return HZState{}, err
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
	var u hzUser
	for cursor.Next(&u) {
		users = append(users, u.Name)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (d *DB) GetProjectsByKey(publicKey string) ([]*types.Project, error) {
	q := projects.GetAllByIndex("Users",
		r.Args(users.GetAllByIndex("PublicSSHKeys", publicKey).
			Field("id").CoerceTo("array")))
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("Couldn't get projectAddrs by key: %v", err)
		return nil, err
	}
	defer cursor.Close()
	var projects []*types.Project
	var p *types.Project
	for cursor.Next(&p) {
		projects = append(projects, p)
		p = nil
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}

func (d *DB) GetProjectsByUsers(users []string) ([]*types.Project, error) {
	q := projects.GetAllByIndex("Users", r.Args(users))
	cursor, err := q.Run(d.session)
	if err != nil {
		d.log.Error("Couldn't get projects by users: %v", err)
		return nil, err
	}
	defer cursor.Close()
	var projects []*types.Project
	var p *types.Project
	for cursor.Next(&p) {
		projects = append(projects, p)
		p = nil
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}

func (d *DB) GetProjectIDByDomain(domainName string) (*types.ProjectID, error) {
	var domain types.Domain
	err := runOne(domains.Get(domainName), d.session, &domain)
	if err != nil {
		if err != r.ErrEmptyResult {
			return nil, err
		}
		return nil, fmt.Errorf("No such domain.")
	}
	return &domain.ProjectID, nil
}

type ProjectChange struct {
	OldVal *types.Project `gorethink:"old_val"`
	NewVal *types.Project `gorethink:"new_val"`
}

func (d *DB) projectChangesLoop(out chan<- ProjectChange) {
	query := projects.Changes(r.ChangesOpts{IncludeInitial: true})
	for {
		ch := make(chan ProjectChange)
		cur, err := query.Run(d.session)
		if err != nil {
			d.log.Error("Couldn't query for config changes: %s", err)
			time.Sleep(time.Second * 5)
			continue
		}
		cur.Listen(ch)
		for el := range ch {
			out <- el
		}
		err = cur.Err()
		d.log.Error("Config changes loop ended: %v", err)
	}
}

func (d *DB) ProjectChanges(out chan<- ProjectChange) {
	go d.projectChangesLoop(out)
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
