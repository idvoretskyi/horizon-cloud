package db

import (
	"fmt"

	r "github.com/dancannon/gorethink"
)

type DB struct {
	session *r.Session
}

var (
	configs = r.DB("test").Table("configs")
)

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

func setBasicError(typeName string, objectName string) error {
	return fmt.Errorf("internal error: unable to set %v `%v`", typeName, objectName)
}

func getBasicError(typeName string, name string) error {
	return fmt.Errorf("internal error: unable to get %v `%s`", typeName, name)
}

func (d *DB) SetConfig(c *Config) error {
	return d.setBasicType(configs, "config", c.Name, c)
}

func (d *DB) GetConfig(name string) (*Config, error) {
	var c Config
	err := d.getBasicType(configs, "config", name, &c)
	return &c, err
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
