package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

// TODO: find a way to figure out which fields were parsed and which
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

func setProjectKubeConfig(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.SetProjectKubeConfigReq
	if !decode(rw, req.Body, &r) {
		return
	}
	project, err := ctx.DB().SetProjectKubeConfig(r.Project, r.KubeConfig)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		api.WriteJSONError(rw, http.StatusInternalServerError,
			fmt.Errorf("Unable to retrieve project."))
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.SetProjectKubeConfigResp{
		Project: *project,
	})
}

func addProjectUsers(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.AddProjectUsersReq
	if !decode(rw, req.Body, &r) {
		return
	}
	project, err := ctx.DB().AddProjectUsers(r.Project, r.Users)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		api.WriteJSONError(rw, http.StatusInternalServerError,
			fmt.Errorf("Unable to retrieve project."))
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.AddProjectUsersResp{
		Project: *project,
	})
}

func getProject(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.GetProjectReq
	if !decode(rw, req.Body, &r) {
		return
	}
	project, err := ctx.DB().GetProject(r.Project)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		api.WriteJSONError(rw, http.StatusInternalServerError,
			fmt.Errorf("Unable to retrieve project."))
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.GetProjectResp{
		Project: *project,
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

func getProjectAddrsByKey(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var gp api.GetProjectAddrsByKeyReq
	if !decode(rw, req.Body, &gp) {
		return
	}
	projectAddrs, err := ctx.DB().GetProjectAddrsByKey(gp.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK,
		api.GetProjectAddrsByKeyResp{ProjectAddrs: projectAddrs})
}

func getProjectAddrByDomain(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.GetProjectAddrByDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	projectAddr, err := ctx.DB().GetProjectAddrByDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK,
		api.GetProjectAddrByDomainResp{ProjectAddr: projectAddr})
}

func maybeUpdateHorizonConfig(
	ctx *hzhttp.Context, project string, hzConf types.HorizonConfig) error {
	// Note: errors from this function are passed to the user.

	ctx = ctx.WithLog(map[string]interface{}{
		"action": "maybeUpdateHorizonConfig",
	})

	newVersion, err := ctx.DB().MaybeUpdateHorizonConfig(project, hzConf)
	ctx.Info("version %v (%v)", newVersion, err)
	if err != nil {
		ctx.Error("Error calling MaybeUpdateHorizonConifg(%v, %v): %v",
			project, hzConf, err)
		return fmt.Errorf("error talking to database")
	}
	// No need to do anything.
	if newVersion == 0 {
		return nil
	}

	hzState, err := ctx.DB().WaitForHorizonConfigVersion(project, newVersion)
	ctx.Info("hzState %v (%v)", hzState, err)
	if err != nil {
		ctx.Error("Error calling WaitForHorizonConfigVersion(%v, %v): %v",
			project, newVersion, err)
		return fmt.Errorf("Error waiting for Horizon Config to be applied.")
	}
	switch hzState.Typ {
	case db.HZError:
		return fmt.Errorf("error applying Horizon config: %v", hzState.LastError)
	case db.HZApplied:
		return nil
	case db.HZSuperseded:
		return fmt.Errorf("Horizon config superseded by later config")
	case db.HZDeleted:
		return fmt.Errorf("Horizon config superseded by project deletion")
	}

	panic("hzState.Typ switch is not exhaustive")
}

func updateProjectManifest(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {

	var r api.UpdateProjectManifestReq
	if !decode(rw, req.Body, &r) {
		return
	}

	ctx = ctx.WithLog(map[string]interface{}{
		"project": util.TrueName(r.Project),
	})

	tokData, err := api.VerifyToken(r.Token, tokenSecret)
	if err != nil {
		err = fmt.Errorf("bad token in request: %v", err)
		ctx.UserError("%v", err)
		api.WriteJSONError(rw, http.StatusBadRequest, err)
		return
	}

	allowedProjectAddrs, err := ctx.DB().GetProjectAddrsByUsers(tokData.Users)
	if err != nil {
		ctx.Error("Couldn't get project list for users: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	found := false
	for _, projectAddr := range allowedProjectAddrs {
		if util.TrueName(projectAddr.Name) == util.TrueName(r.Project) {
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

	go projectSync(baseCtx)

	paths := []struct {
		Path          string
		Func          func(ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request)
		RequireSecret bool
	}{
		// Client uses these.
		{api.UpdateProjectManifestPath, updateProjectManifest, false},

		// Web interface uses these.
		{api.GetProjectPath, getProject, true},
		{api.SetProjectKubeConfigPath, setProjectKubeConfig, true},
		{api.AddProjectUsersPath, addProjectUsers, true},
		{api.UserCreatePath, userCreate, true},
		{api.UserGetPath, userGet, true},
		{api.UserAddKeysPath, userAddKeys, true},
		{api.UserDelKeysPath, userDelKeys, true},
		{api.SetDomainPath, setDomain, true},
		{api.GetDomainsByProjectPath, getDomainsByProject, true},

		// Other server stuff uses these.
		{api.GetUsersByKeyPath, getUsersByKey, true},
		{api.GetProjectAddrsByKeyPath, getProjectAddrsByKey, true},
		{api.GetProjectAddrByDomainPath, getProjectAddrByDomain, true},
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
