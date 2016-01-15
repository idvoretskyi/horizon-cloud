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

func createConfigError(name string) error {
	return fmt.Errorf("internal error: unable to create config `%s`", name)
}

func getConfigError(name string) error {
	return fmt.Errorf("internal error: unable to get config `%s`", name)
}

func (d *DB) CreateConfig(c *api.Config) error {
	res, err := configs.Insert(c).RunWrite(d.session)
	if err != nil {
		// RSI: log
		// RSI: detect and report "already exists error" specially
		return createConfigError(c.Name)
	}
	if res.Inserted != 1 {
		// RSI: serious log
		return createConfigError(c.Name)
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
