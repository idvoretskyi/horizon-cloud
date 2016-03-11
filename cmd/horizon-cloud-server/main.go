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
	var dc api.DesiredConfig
	if !decode(rw, req.Body, &dc) {
		return
	}
	if err := rdb.SetConfig(*api.ConfigFromDesired(&dc)); err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	// RSI: instead read the actual new config out of the database.
	api.WriteJSONResp(rw, http.StatusOK, api.ConfigFromDesired(&dc))
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

func getProjects(rw http.ResponseWriter, req *http.Request) {
	var gp api.GetProjectsReq
	if !decode(rw, req.Body, &gp) {
		return
	}
	projects, err := rdb.GetProjects(gp.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectsResp{Projects: projects})
}

func getByAlias(rw http.ResponseWriter, req *http.Request) {
	var gp api.GetByAliasReq
	if !decode(rw, req.Body, &gp) {
		return
	}
	project, err := rdb.GetByAlias(gp.Alias)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetByAliasResp{Project: project})
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
	sshServer            = "130.211.131.118"
	sshServerFingerprint = `130.211.131.118 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCfQJqUbNs6n1r0BtWeODDlB3fXUX0/iE+m7KfkkQXMxr7+Bmjz/Tl91NZIch09NozfenYV6IVdamFMdwSDau5nt5/VPd/QuxDUCeXBvB8XOfUw4Arwew4wQMTU27NqngI0FIYbkZw2T7zMDfocLBhwJh7Ms8bJwGezZ9oYKCGuFvvUMMNmrbKTa/SoF4PY1XPXQOXJdry8oyHsWETcr2BT0qWS+3uoG1ipui/LfeVq6A1M71IT/BVjaGQWm+l8T+vJYUQqLgQYc8qKvmA2S/YGqRv87L9W8jhO6lIFMvWvCsQ7ppuLCDIz0DubP6gD0Lj8piI+IcVD7fuMfGOLQo17`
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

	handlers := []struct {
		Path          string
		Func          func(w http.ResponseWriter, r *http.Request)
		RequireSecret bool
	}{
		{"/v1/config/set", setConfig, false},
		{"/v1/config/get", getConfig, false},
		{"/v1/config/ensure_connectable", ensureConfigConnectable, false},
		{"/v1/config/wait_applied", waitConfigApplied, false},
		{"/v1/projects/get", getProjects, true},
		{"/v1/projects/getByAlias", getByAlias, true},
	}

	for _, handler := range handlers {
		var h http.Handler = http.HandlerFunc(handler.Func)
		if handler.RequireSecret {
			h = api.RequireSecret(sharedSecret, h)
		}
		http.Handle(handler.Path, h)
	}

	log.Printf("Started.")
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
