package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/db"
)

// RSI: find a way to figure out which fields were parsed and which
// were defaulted so that we can error if we get sent incomplete
// messages.

// RSI: make this non-global.
var rdb *db.DB

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

func setConfig(rw http.ResponseWriter, req *http.Request) {
	var r api.SetConfigReq
	if !decode(rw, req.Body, &r) {
		return
	}
	newConf, err := rdb.SetConfig(*api.ConfigFromDesired(&r.DesiredConfig))
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.SetConfigResp{*newConf})
}

func getConfig(rw http.ResponseWriter, req *http.Request) {
	var gc api.GetConfigReq
	if !decode(rw, req.Body, &gc) {
		return
	}
	// RSI(sec): don't let people read other people's configs.
	config, err := rdb.GetConfig(gc.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetConfigResp{
		Config: *config,
	})
}

func userCreate(rw http.ResponseWriter, req *http.Request) {
	var r api.UserCreateReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := rdb.UserCreate(r.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserCreateResp{})
}

func userGet(rw http.ResponseWriter, req *http.Request) {
	var r api.UserGetReq
	if !decode(rw, req.Body, &r) {
		return
	}
	user, err := rdb.UserGet(r.Name)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserGetResp{User: *user})
}

func userAddKeys(rw http.ResponseWriter, req *http.Request) {
	var r api.UserAddKeysReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := rdb.UserAddKeys(r.Name, r.Keys)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserAddKeysResp{})
}

func userDelKeys(rw http.ResponseWriter, req *http.Request) {
	var r api.UserDelKeysReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := rdb.UserDelKeys(r.Name, r.Keys)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.UserDelKeysResp{})
}

func setDomain(rw http.ResponseWriter, req *http.Request) {
	var r api.SetDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := rdb.SetDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.SetDomainResp{})
}

func getDomainsByProject(rw http.ResponseWriter, req *http.Request) {
	var r api.GetDomainsByProjectReq
	if !decode(rw, req.Body, &r) {
		return
	}
	domains, err := rdb.GetDomainsByProject(r.Project)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetDomainsByProjectResp{domains})
}

func getProjectsByKey(rw http.ResponseWriter, req *http.Request) {
	var gp api.GetProjectsByKeyReq
	if !decode(rw, req.Body, &gp) {
		return
	}
	projects, err := rdb.GetProjectsByKey(gp.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectsByKeyResp{Projects: projects})
}

func getProjectByDomain(rw http.ResponseWriter, req *http.Request) {
	var r api.GetProjectByDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	project, err := rdb.GetByDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectByDomainResp{project})
}

func ensureConfigConnectable(rw http.ResponseWriter, req *http.Request) {
	var creq api.EnsureConfigConnectableReq
	if !decode(rw, req.Body, &creq) {
		return
	}
	// RSI(sec): don't let people read other people's configs.
	config, err := rdb.EnsureConfigConnectable(
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
	sshServer            = "104.197.227.19"
	sshServerFingerprint = `104.197.227.19 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCfQJqUbNs6n1r0BtWeODDlB3fXUX0/iE+m7KfkkQXMxr7+Bmjz/Tl91NZIch09NozfenYV6IVdamFMdwSDau5nt5/VPd/QuxDUCeXBvB8XOfUw4Arwew4wQMTU27NqngI0FIYbkZw2T7zMDfocLBhwJh7Ms8bJwGezZ9oYKCGuFvvUMMNmrbKTa/SoF4PY1XPXQOXJdry8oyHsWETcr2BT0qWS+3uoG1ipui/LfeVq6A1M71IT/BVjaGQWm+l8T+vJYUQqLgQYc8qKvmA2S/YGqRv87L9W8jhO6lIFMvWvCsQ7ppuLCDIz0DubP6gD0Lj8piI+IcVD7fuMfGOLQo17`
)

func waitConfigApplied(rw http.ResponseWriter, req *http.Request) {
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

	config, err := rdb.WaitConfigApplied(wca.Name, wca.Version, cancel)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}

	api.WriteJSONResp(rw, http.StatusOK, api.WaitConfigAppliedResp{
		Config: *config,
		Target: api.Target{
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

	rdb, err = db.New()
	if err != nil {
		log.Fatal("unable to connect to RethinkDB: ", err)
	}
	go configSync(rdb)

	paths := []struct {
		Path          string
		Func          func(w http.ResponseWriter, r *http.Request)
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

	mux := http.NewServeMux()
	for _, path := range paths {
		var h http.Handler = http.HandlerFunc(path.Func)
		if path.RequireSecret {
			h = api.RequireSecret(sharedSecret, h)
		}
		mux.Handle(path.Path, h)
	}
	logMux := handlers.LoggingHandler(os.Stdout, mux)

	log.Printf("Started.")
	log.Fatal(http.ListenAndServe(*listenAddr, logMux))
}
