package db

import (
	"fmt"

	r "github.com/dancannon/gorethink"
	"github.com/rethinkdb/fusion-ops/api"
)

type DB struct {
	session *r.Session
}

var configs = r.DB("test").Table("configs")

func New() (*DB, error) {
	session, err := r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "test",
		MaxIdle:  10,
		MaxOpen:  10,
	})
	if err != nil {
		return nil, err
	}
	return &DB{session}, nil
}

func setConfigError(name string) error {
	return fmt.Errorf("internal error: unable to set config `%s`", name)
}

func getConfigError(name string) error {
	return fmt.Errorf("internal error: unable to get config `%s`", name)
}

func (d *DB) SetConfig(c *api.Config) error {
	res, err := configs.Insert(c, r.InsertOpts{Conflict: "replace"}).RunWrite(d.session)
	if err != nil {
		// RSI: log
		return setConfigError(c.Name)
	}
	if res.Inserted+res.Unchanged+res.Replaced != 1 {
		// RSI: serious log
		return setConfigError(c.Name)
	}
	return nil
}

func (d *DB) GetConfig(name string) (*api.Config, error) {
	res, err := configs.Get(name).Run(d.session)
	if err != nil {
		return nil, getConfigError(name)
	}
	var c api.Config
	err = res.One(&c)
	if err == r.ErrEmptyResult {
		return nil, fmt.Errorf("config `%s` does not exist", name)
	} else if err != nil {
		return nil, getConfigError(name)
	}
	return &c, nil
}
