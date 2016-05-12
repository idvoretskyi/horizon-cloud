package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/cloud/storage"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/db"
	"github.com/rethinkdb/horizon-cloud/internal/gcloud"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/rethinkdb/horizon-cloud/internal/kube"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
)

// RSI: find a way to figure out which fields were parsed and which
// were defaulted so that we can error if we get sent incomplete
// messages.

var (
	clusterName   string
	templatePath  string
	storageBucket string
	tokenSecret   []byte
)

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
	var r api.GetConfigReq
	if !decode(rw, req.Body, &r) {
		return
	}
	config, err := ctx.DB().GetConfig(r.Name)
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

func getUsersByKey(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var gu api.GetUsersByKeyReq
	if !decode(rw, req.Body, &gu) {
		return
	}
	users, err := ctx.DB().GetUsersByKey(gu.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetUsersByKeyResp{Users: users})
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

func ensureConfigConnectable(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
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

// Note: errors from this function are passed to the user.
func maybeUpdateHorizonConfig(
	ctx *hzhttp.Context, project string, hzConf []byte) error {

	// This has to be stored in a variable because Go refuses to let you
	// slice a temporary, DESPITE THE FACT THAT IT IS A FUCKING
	// GARBAGE-COLLECTED LANGUAGE THAT IS DAMN WELL CAPABLE OF EXTENDING
	// THE LIFETIME OF SAID TEMPORARY AS LONG AS NECESSARY IF IT WASN'T
	// TOO LAZY AND INCONSISTENT TO FUCKING BOTHER.
	shaBytes := sha256.Sum256(hzConf)
	confHash := hex.EncodeToString(shaBytes[:])
	matches, err := ctx.DB().HorizonConfigHashMatches(project, confHash)
	if err != nil {
		ctx.Error("Error calling hzConfHashmatches(%s, %v): %v", project, confHash, err)
		return fmt.Errorf("Error accessing existing project configuration.")
	}
	if matches {
		return nil
	}

	// Update the configuration.
	k := ctx.Kube()
	pods, err := k.GetHorizonPodsForProject(project)
	if err != nil {
		ctx.Error("Error calling GetHorizonPodsForProject(%s): %v", project, err)
		return fmt.Errorf("Error accessing horizon instances for project `%s`.", project)
	}
	if len(pods) == 0 {
		err = fmt.Errorf("No pods found for project `%s`.", project)
		ctx.Error("%v", err)
		return err
	}

	pod := pods[rand.Intn(len(pods))]
	stdout, stderr, err := k.Exec(kube.ExecOptions{
		PodName: pod,
		In:      bytes.NewReader(hzConf),
		Command: []string{"su", "-s", "/bin/sh", "horizon", "-c",
			"sleep 0.3; cat > /tmp/conf; echo stdout; echo stderr >&2"},
	})
	if err != nil {
		err = fmt.Errorf("Error setting Horizon config:\n"+
			"\nStdout:\n%s\n"+
			"\nStderr:\n%s\n"+
			"\nError:\n%v\n", stdout, stderr, err)
		ctx.Error("%v", err)
		return err
	}

	err = ctx.DB().SetHorizonConfigHash(project, confHash)

	return nil
}

func updateProjectManifest(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.UpdateProjectManifestReq
	if !decode(rw, req.Body, &r) {
		return
	}

	tokData, err := api.VerifyToken(r.Token, tokenSecret)
	if err != nil {
		err = fmt.Errorf("bad token in request: %v", err)
		ctx.UserError("%v", err)
		api.WriteJSONError(rw, http.StatusBadRequest, err)
		return
	}

	allowedProjects, err := ctx.DB().GetProjectsByUsers(tokData.Users)
	if err != nil {
		ctx.Error("Couldn't get project list for users: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	found := false
	for _, proj := range allowedProjects {
		if util.TrueName(proj.Name) == util.TrueName(r.Project) {
			found = true
			break
		}
	}

	if !found {
		ctx.UserError(
			"User %v not allowed to deploy to project %v", tokData.Users, r.Project)
		api.WriteJSONError(rw, http.StatusBadRequest,
			errors.New("You are not allowed to deploy to that project"))
		return
	}

	// TODO: generalize
	gc, err := gcloud.New(ctx.ServiceAccount(), clusterName, "us-central1-f")
	if err != nil {
		ctx.Error("Couldn't create gcloud instance: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	stagingPrefix := "deploy/" + util.TrueName(r.Project) + "/staging/"

	requests, err := requestsForFilelist(
		ctx,
		gc.StorageClient(),
		storageBucket,
		stagingPrefix,
		r.Files)
	if err != nil {
		ctx.Error("Couldn't create request list for file list: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	if len(requests) > 0 {
		api.WriteJSONResp(rw, http.StatusOK, api.UpdateProjectManifestResp{
			NeededRequests: requests,
		})
		return
	}

	// If we get here, the user has successfully uploaded all the files
	// they need to upload.

	err = maybeUpdateHorizonConfig(ctx, r.Project, r.HorizonConfig)
	if err != nil {
		ctx.Error("Unable to update Horizon config: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			fmt.Errorf("Unable to update Horizon config: %v", err))
		return
	}

	err = copyAllObjects(ctx, gc.StorageClient(), storageBucket,
		"horizon/", stagingPrefix+"horizon/")
	if err != nil {
		ctx.Error("Couldn't copy horizon objects: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	domains, err := ctx.DB().GetDomainsByProject(r.Project)
	if err != nil {
		ctx.Error("Couldn't get domains for %v: %v", r.Project, err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	for _, domain := range domains {
		err := copyAllObjects(
			ctx,
			gc.StorageClient(),
			storageBucket,
			stagingPrefix,
			"domains/"+domain+"/")
		if err != nil {
			ctx.Error("Couldn't copy objects for %v to domains/%v: %v",
				r.Project, domain, err)
			api.WriteJSONError(rw, http.StatusInternalServerError,
				errors.New("Internal error"))
			return
		}
	}

	api.WriteJSONResp(rw, http.StatusOK, api.UpdateProjectManifestResp{
		NeededRequests: []types.FileUploadRequest{},
	})
}

const (
	sshServer            = "ssh.hzc.io"
	sshServerFingerprint = `ssh.hzc.io ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCfQJqUbNs6n1r0BtWeODDlB3fXUX0/iE+m7KfkkQXMxr7+Bmjz/Tl91NZIch09NozfenYV6IVdamFMdwSDau5nt5/VPd/QuxDUCeXBvB8XOfUw4Arwew4wQMTU27NqngI0FIYbkZw2T7zMDfocLBhwJh7Ms8bJwGezZ9oYKCGuFvvUMMNmrbKTa/SoF4PY1XPXQOXJdry8oyHsWETcr2BT0qWS+3uoG1ipui/LfeVq6A1M71IT/BVjaGQWm+l8T+vJYUQqLgQYc8qKvmA2S/YGqRv87L9W8jhO6lIFMvWvCsQ7ppuLCDIz0DubP6gD0Lj8piI+IcVD7fuMfGOLQo17`
)

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
		"shared_secret",
		"/secrets/api-shared-secret/api-shared-secret",
		"Location of API shared secret",
	)

	tokenSecretFile := flag.String(
		"token_secret",
		"/secrets/token-secret/token-secret",
		"Location of token secret file",
	)

	flag.StringVar(&clusterName, "cluster_name", "horizon-cloud-1239",
		"Name of the GCE cluster to use.")

	flag.StringVar(&templatePath, "template_path",
		os.Getenv("HOME")+"/go/src/github.com/rethinkdb/horizon-cloud/templates/",
		"Path to the templates to use when creating Kube objects.")

	flag.StringVar(&storageBucket, "storage_bucket",
		"hzc-dev-io-userdata",
		"Storage bucket to write user objects to")

	serviceAccountFile := flag.String(
		"service_account",
		"/secrets/gcloud-service-account/gcloud-service-account.json",
		"Path to the JSON service account.")

	flag.Parse()

	data, err := ioutil.ReadFile(*sharedSecretFile)
	if err != nil {
		log.Fatal("Unable to read shared secret file: ", err)
	}
	if len(data) < 16 {
		log.Fatal("Shared secret was not long enough")
	}
	sharedSecret := string(data)

	tokenSecret, err = ioutil.ReadFile(*tokenSecretFile)
	if err != nil {
		log.Fatal("Unable to read token secret file: ", err)
	}
	if len(tokenSecret) < 16 {
		log.Fatal("Token secret was not long enough")
	}

	rdbConn, err := db.New()
	if err != nil {
		log.Fatal("Unable to connect to RethinkDB: ", err)
	}
	baseCtx = baseCtx.WithDBConnection(rdbConn)

	serviceAccountData, err := ioutil.ReadFile(*serviceAccountFile)
	if err != nil {
		log.Fatal("Unable to read service account file: ", err)
	}
	serviceAccount, err := google.JWTConfigFromJSON(serviceAccountData, storage.ScopeFullControl, compute.ComputeScope)
	if err != nil {
		log.Fatal("Unable to parse service account: ", err)
	}
	baseCtx = baseCtx.WithServiceAccount(serviceAccount)

	region := "us-central1-f"
	gc, err := gcloud.New(serviceAccount, clusterName, region)
	if err != nil {
		log.Fatal("Unable to create gcloud client: ", err)
	}

	k := kube.New(templatePath, gc)
	baseCtx = baseCtx.WithKube(k)

	go configSync(baseCtx)

	paths := []struct {
		Path          string
		Func          func(ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request)
		RequireSecret bool
	}{
		// Client uses these.
		{api.EnsureConfigConnectablePath, ensureConfigConnectable, false},
		{api.UpdateProjectManifestPath, updateProjectManifest, false},

		// Mike uses these.
		{api.SetConfigPath, setConfig, true},
		{api.GetConfigPath, getConfig, true},
		{api.UserCreatePath, userCreate, true},
		{api.UserGetPath, userGet, true},
		{api.UserAddKeysPath, userAddKeys, true},
		{api.UserDelKeysPath, userDelKeys, true},
		{api.SetDomainPath, setDomain, true},
		{api.GetDomainsByProjectPath, getDomainsByProject, true},

		// Other server stuff uses these.
		{api.GetUsersByKeyPath, getUsersByKey, true},
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
