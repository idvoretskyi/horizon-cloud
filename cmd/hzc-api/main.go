package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/types"
)

// RSI: find a way to figure out which fields were parsed and which
// were defaulted so that we can error if we get sent incomplete
// messages.

type validator interface {
	Validate() error
}

func decode(rw http.ResponseWriter, r io.Reader, body validator) bool {
	if err := json.NewDecoder(r).Decode(body); err != nil {
		api.WriteJSONError(rw, http.StatusBadRequest, err)
		return false
	}
	if err := body.Validate(); err != nil {
		api.WriteJSONError(rw, http.StatusBadRequest, err)
		return false
	}
	return true
}

func setConfig(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.SetConfigReq
	if !decode(rw, req.Body, &r) {
		return
	}
	newConf, err := ctx.DB().SetConfig(*types.ConfigFromDesired(&r.DesiredConfig))
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.SetConfigResp{*newConf})
}

func getConfig(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var gc api.GetConfigReq
	if !decode(rw, req.Body, &gc) {
		return
	}
	// RSI(sec): don't let people read other people's configs.
	config, err := ctx.DB().GetConfig(gc.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetConfigResp{
		Config: *config,
	})
}

func userCreate(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.UserCreateReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := ctx.DB().UserCreate(r.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserCreateResp{})
}

func userGet(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.UserGetReq
	if !decode(rw, req.Body, &r) {
		return
	}
	user, err := ctx.DB().UserGet(r.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserGetResp{User: *user})
}

func userAddKeys(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.UserAddKeysReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := ctx.DB().UserAddKeys(r.Name, r.Keys)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserAddKeysResp{})
}

func userDelKeys(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.UserDelKeysReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := ctx.DB().UserDelKeys(r.Name, r.Keys)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserDelKeysResp{})
}

func setDomain(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.SetDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := ctx.DB().SetDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.SetDomainResp{})
}

func getDomainsByProject(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.GetDomainsByProjectReq
	if !decode(rw, req.Body, &r) {
		return
	}
	domains, err := ctx.DB().GetDomainsByProject(r.Project)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetDomainsByProjectResp{domains})
}

func getProjectsByKey(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var gp api.GetProjectsByKeyReq
	if !decode(rw, req.Body, &gp) {
		return
	}
	projects, err := ctx.DB().GetProjectsByKey(gp.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectsByKeyResp{Projects: projects})
}

func getProjectByDomain(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.GetProjectByDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	project, err := ctx.DB().GetByDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectByDomainResp{project})
}

func ensureConfigConnectable(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var creq api.EnsureConfigConnectableReq
	if !decode(rw, req.Body, &creq) {
		return
	}
	// RSI(sec): don't let people read other people's configs.
	config, err := ctx.DB().EnsureConfigConnectable(
		creq.Name, creq.AllowClusterStart)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.EnsureConfigConnectableResp{
		Config: *config,
	})
}

const (
	sshServer            = "ssh.hzc.io"
	sshServerFingerprint = `ssh.hzc.io ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCfQJqUbNs6n1r0BtWeODDlB3fXUX0/iE+m7KfkkQXMxr7+Bmjz/Tl91NZIch09NozfenYV6IVdamFMdwSDau5nt5/VPd/QuxDUCeXBvB8XOfUw4Arwew4wQMTU27NqngI0FIYbkZw2T7zMDfocLBhwJh7Ms8bJwGezZ9oYKCGuFvvUMMNmrbKTa/SoF4PY1XPXQOXJdry8oyHsWETcr2BT0qWS+3uoG1ipui/LfeVq6A1M71IT/BVjaGQWm+l8T+vJYUQqLgQYc8qKvmA2S/YGqRv87L9W8jhO6lIFMvWvCsQ7ppuLCDIz0DubP6gD0Lj8piI+IcVD7fuMfGOLQo17`
)

func waitConfigApplied(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var wca api.WaitConfigAppliedReq
	if !decode(rw, req.Body, &wca) {
		return
	}

	// RSI: access limitations a la getConfig

	returned := make(chan struct{})
	defer close(returned)

	var closeNotify <-chan bool
	if cnrw, ok := rw.(http.CloseNotifier); ok {
		closeNotify = cnrw.CloseNotify()
	}

	cancel := make(chan struct{})
	go func() {
		select {
		case <-returned:
			// do nothing
		case <-closeNotify:
			close(cancel)
		}
	}()

	config, err := ctx.DB().WaitConfigApplied(wca.Name, wca.Version, cancel)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	api.WriteJSONResp(rw, http.StatusOK, api.WaitConfigAppliedResp{
		Config: *config,
		Target: types.Target{
			Hostname:     sshServer,
			Fingerprints: []string{sshServerFingerprint},
			Username:     "horizon",
			DeployDir:    "/data/",
			DeployCmd:    "/home/horizon/post-deploy.sh",
		},
	})
}

func main() {
	log.SetFlags(log.Lshortfile)

	logger, err := hzlog.MainLogger("hzc-api")
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(hzlog.WriterLogger(logger))

	baseCtx := hzhttp.NewContext(logger)

	listenAddr := flag.String("listen", ":8000", "HTTP listening address")
	sharedSecretFile := flag.String(
		"shared-secret",
		"/secrets/api-shared-secret/api-shared-secret",
		"Location of API shared secret",
	)
	flag.Parse()

	data, err := ioutil.ReadFile(*sharedSecretFile)
	if err != nil {
		log.Fatal("Unable to read shared secret file: ", err)
	}
	if len(data) < 16 {
		log.Fatal("Shared secret was not long enough")
	}
	sharedSecret := string(data)

	rdbConn, err := db.New()
	if err != nil {
		log.Fatal("Unable to connect to RethinkDB: ", err)
	}
	baseCtx = baseCtx.WithDBConnection(rdbConn)

	go configSync(baseCtx.DB())

	paths := []struct {
		Path          string
		Func          func(ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request)
		RequireSecret bool
	}{
		{api.EnsureConfigConnectablePath, ensureConfigConnectable, false},
		{api.WaitConfigAppliedPath, waitConfigApplied, false},

		// Mike uses these.
		{api.SetConfigPath, setConfig, true},
		{api.GetConfigPath, getConfig, true},
		{api.UserCreatePath, userCreate, true},
		{api.UserGetPath, userGet, true},
		{api.UserAddKeysPath, userAddKeys, true},
		{api.UserDelKeysPath, userDelKeys, true},
		{api.SetDomainPath, setDomain, true},
		{api.GetDomainsByProjectPath, getDomainsByProject, true},

		// Chris uses these.
		{api.GetProjectsByKeyPath, getProjectsByKey, true},
		{api.GetProjectByDomainPath, getProjectByDomain, true},
	}

	mux := hzhttp.NewMuxer()
	for _, path := range paths {
		var h hzhttp.Handler = hzhttp.HandlerFunc(path.Func)
		if path.RequireSecret {
			h = api.RequireSecret(sharedSecret, h)
		}
		mux.RegisterPath(path.Path, h)
	}
	logMux := hzhttp.LogHTTPRequests(mux)

	logger.Info("Started.")
	err = http.ListenAndServe(*listenAddr, hzhttp.BaseContext(baseCtx, logMux))
	if err != nil {
		logger.Error("Couldn't serve on %v: %v", *listenAddr, err)
	}
}
