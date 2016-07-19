package hzhttp

import (
	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
	"golang.org/x/oauth2/jwt"
)

// A Context is an immutable source of information for a particular HTTP request.
type Context struct {
	LogContext     *hzlog.Logger
	DBConn         *db.DBConnection
	ServiceAccount *jwt.Config
	Kube           *kube.Kube
}

// NewContext returns a new Context.
func NewContext(logcontext *hzlog.Logger) *Context {
	if logcontext == nil {
		logcontext = hzlog.BlankLogger()
	}
	return &Context{
		LogContext: logcontext,
	}
}

// EmptyLog logs the current key-value pairs with no message or level.
func (c *Context) EmptyLog() {
	c.LogContext.OutputDepth(2)
}

// Info logs the current key-value pairs with `level` set to `"info"` and
// `message` to the result of `fmt.Sprintf` on the given arguments.
func (c *Context) Info(format string, args ...interface{}) {
	c.LogContext.InfoDepth(2, format, args...)
}

// Error logs the current key-value pairs with `level` set to `"info"` and
// `message` set to the result of `fmt.Sprintf` on the given arguments.
func (c *Context) Error(format string, args ...interface{}) {
	c.LogContext.ErrorDepth(2, format, args...)
}

func (c *Context) UserError(format string, args ...interface{}) {
	c.LogContext.UserErrorDepth(2, format, args...)
}

func (c *Context) MaybeError(err error) {
	if err != nil {
		c.Error("%v", err)
	}
}

// Log logs the current key-value pairs with `message` set to the result of
// `fmt.Sprintf` on the given arguments.
func (c *Context) Log(format string, args ...interface{}) {
	c.LogContext.LogDepth(2, format, args...)
}

// WithLog returns a new Context with the given key-value pairs added to the
// LogContext.
func (c *Context) WithLog(m map[string]interface{}) *Context {
	c2 := *c
	c2.LogContext = c.LogContext.With(m)
	return &c2
}

func (c *Context) WithParts(cpart *Context) *Context {
	out := *c
	if cpart.LogContext != nil {
		panic("Use WithLog, not WithParts to edit the log context")
	}
	if cpart.DBConn != nil {
		out.DBConn = cpart.DBConn
	}
	if cpart.ServiceAccount != nil {
		out.ServiceAccount = cpart.ServiceAccount
	}
	if cpart.Kube != nil {
		out.Kube = cpart.Kube
	}
	return &out
}

func (c *Context) DB() *db.DB {
	return c.DBConn.WithLogger(c.LogContext)
}
