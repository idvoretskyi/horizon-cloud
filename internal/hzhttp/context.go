package hzhttp

import (
	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
)

// A Context is an immutable source of information for a particular HTTP request.
type Context struct {
	logContext *hzlog.Logger
	dbconn     *db.DBConnection
}

// NewContext returns a new Context.
func NewContext(logcontext *hzlog.Logger) *Context {
	if logcontext == nil {
		logcontext = hzlog.BlankLogger()
	}
	return &Context{
		logContext: logcontext,
	}
}

// EmptyLog logs the current key-value pairs with no message or level.
func (c *Context) EmptyLog() {
	c.logContext.OutputDepth(2)
}

// Info logs the current key-value pairs with `level` set to `"info"` and
// `message` to the result of `fmt.Sprintf` on the given arguments.
func (c *Context) Info(format string, args ...interface{}) {
	c.logContext.InfoDepth(2, format, args...)
}

// Error logs the current key-value pairs with `level` set to `"info"` and
// `message` set to the result of `fmt.Sprintf` on the given arguments.
func (c *Context) Error(format string, args ...interface{}) {
	c.logContext.ErrorDepth(2, format, args...)
}

func (c *Context) UserError(format string, args ...interface{}) {
	c.logContext.UserErrorDepth(2, format, args...)
}

// Log logs the current key-value pairs with `message` set to the result of
// `fmt.Sprintf` on the given arguments.
func (c *Context) Log(format string, args ...interface{}) {
	c.logContext.LogDepth(2, format, args...)
}

// WithLog returns a new Context with the given key-value pairs added to the
// LogContext.
func (c *Context) WithLog(m map[string]interface{}) *Context {
	c2 := *c
	c2.logContext = c.logContext.With(m)
	return &c2
}

func (c *Context) WithDBConnection(dbconn *db.DBConnection) *Context {
	c2 := *c
	c2.dbconn = dbconn
	return &c2
}

func (c *Context) DB() *db.DB {
	return c.dbconn.WithLogger(c.logContext)
}