package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
)

// TODO: find a way to figure out which fields were parsed and which
// were defaulted so that we can error if we get sent incomplete
// messages.

var (
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

func delDomain(ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.DelDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	err := ctx.DB().DelDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	api.WriteJSONResp(rw, http.StatusOK, api.DelDomainResp{})
}

func getDomainsByProject(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
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
	projects, err := ctx.DB().GetProjectsByKey(gp.PublicKey)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	addrs := make([]types.ProjectAddr, len(projects))
	for i, p := range projects {
		addrs[i] = p.Addr(storageBucket)
	}
	api.WriteJSONResp(rw, http.StatusOK,
		api.GetProjectAddrsByKeyResp{ProjectAddrs: addrs})
}

func getProjectAddrByDomain(
	ctx *hzhttp.Context, rw http.ResponseWriter, req *http.Request) {
	var r api.GetProjectAddrByDomainReq
	if !decode(rw, req.Body, &r) {
		return
	}
	id, err := ctx.DB().GetProjectIDByDomain(r.Domain)
	if err != nil {
		api.WriteJSONError(rw, http.StatusInternalServerError, err)
		return
	}
	addr := id.Addr(storageBucket)
	api.WriteJSONResp(rw, http.StatusOK,
		api.GetProjectAddrByDomainResp{ProjectAddr: &addr})
}

func maybeUpdateHorizonConfig(
	ctx *hzhttp.Context, projectID types.ProjectID, hzConf types.HorizonConfig) error {
	// Note: errors from this function are passed to the user.

	ctx = ctx.WithLog(map[string]interface{}{
		"action": "maybeUpdateHorizonConfig",
	})

	newVersion, versionErr, err := ctx.DB().MaybeUpdateHorizonConfig(projectID, hzConf)
	ctx.Info("version %v (err version: %v)", newVersion, err)
	if err != nil {
		ctx.Error("Error calling MaybeUpdateHorizonConfig(%v, %v): %v",
			projectID, hzConf, err)
		return fmt.Errorf("error talking to database")
	}
	if versionErr != "" {
		ctx.Error("Version error calling MaybeUpdateHorizonConifg(%v, %v): %v",
			projectID, hzConf, err)
		return errors.New(versionErr)
	}
	// No need to do anything.
	if newVersion == 0 {
		return nil
	}

	hzState, err := ctx.DB().WaitForHorizonConfigVersion(projectID, newVersion)
	ctx.Info("hzState %v (%v)", hzState, err)
	if err != nil {
		ctx.Error("Error calling WaitForHorizonConfigVersion(%v, %v): %v",
			projectID, newVersion, err)
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

	ctx = ctx.WithLog(map[string]interface{}{"project": r.ProjectID})

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
	for _, project := range allowedProjects {
		if r.ProjectID == project.ID {
			found = true
			break
		}
	}

	if !found {
		ctx.UserError(
			"User %v not allowed to deploy to project %v", tokData.Users, r.ProjectID)
		api.WriteJSONError(rw, http.StatusBadRequest,
			errors.New("You are not allowed to deploy to that project"))
		return
	}

	stagingPrefix := "deploy/" + r.ProjectID.KubeName() + "/staging/"

	requests, err := requestsForFilelist(
		ctx,
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

	err = maybeUpdateHorizonConfig(ctx, r.ProjectID, r.HorizonConfig)
	if err != nil {
		ctx.Error("Unable to update Horizon config: %v", err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			fmt.Errorf("Unable to update Horizon config: %v", err))
		return
	}

	err = copyAllObjects(
		ctx,
		storageBucket, stagingPrefix,
		storageBucket, "deploy/"+r.ProjectID.KubeName()+"/active/")
	if err != nil {
		ctx.Error("Couldn't copy objects for %v to active location: %v",
			r.ProjectID, err)
		api.WriteJSONError(rw, http.StatusInternalServerError,
			errors.New("Internal error"))
		return
	}

	api.WriteJSONResp(rw, http.StatusOK, api.UpdateProjectManifestResp{
		NeededRequests: []types.FileUploadRequest{},
	})
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

var RootCmd = &cobra.Command{
	Use:   "hzc-client",
	Short: "Horizon Cloud Client",
	Long:  `A client for accessing Horizon Cloud.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(log.Lshortfile)

		logger, err := hzlog.MainLogger("hzc-api")
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(hzlog.WriterLogger(logger))

		baseCtx := hzhttp.NewContext(logger)

		data, err := ioutil.ReadFile(viper.GetString("shared_secret"))
		if err != nil {
			log.Fatal("Unable to read shared secret file: ", err)
		}
		if len(data) < 16 {
			log.Fatal("Shared secret was not long enough")
		}
		sharedSecret := string(data)

		tokenSecret, err = ioutil.ReadFile(viper.GetString("token_secret"))
		if err != nil {
			log.Fatal("Unable to read token secret file: ", err)
		}
		if len(tokenSecret) < 16 {
			log.Fatal("Token secret was not long enough")
		}

		rdbConn, err := db.New(viper.GetString("rethinkdb_addr"))
		if err != nil {
			log.Fatal("Unable to connect to RethinkDB: ", err)
		}
		baseCtx = baseCtx.WithParts(&hzhttp.Context{DBConn: rdbConn})

		serviceAccountData, err := ioutil.ReadFile(viper.GetString("service_account"))
		if err != nil {
			log.Fatal("Unable to read service account file: ", err)
		}
		serviceAccount, err := google.JWTConfigFromJSON(serviceAccountData, storage.ScopeFullControl, compute.ComputeScope)
		if err != nil {
			log.Fatal("Unable to parse service account: ", err)
		}
		baseCtx = baseCtx.WithParts(&hzhttp.Context{ServiceAccount: serviceAccount})

		storageBucketBytes, err := ioutil.ReadFile(viper.GetString("storage_bucket_file"))
		if err != nil {
			log.Fatal("Unable to read storage bucket file: ", err)
		}
		storageBucket = string(storageBucketBytes)

		region := "us-central1-f" // TODO: Generalize/parameterize
		gc, err := gcloud.New(serviceAccount, viper.GetString("cluster_name"), region)
		if err != nil {
			log.Fatal("Unable to create gcloud client: ", err)
		}
		baseCtx = baseCtx.WithParts(&hzhttp.Context{GCloud: gc})

		k := kube.New(viper.GetString("template_path"),
			viper.GetString("kube_namespace"), gc)
		baseCtx = baseCtx.WithParts(&hzhttp.Context{Kube: k})

		go projectSync(baseCtx)

		paths := []struct {
			Path          string
			Func          func(ctx *hzhttp.Context, w http.ResponseWriter, r *http.Request)
			RequireSecret bool
		}{
			// Client uses these.
			{api.UpdateProjectManifestPath, updateProjectManifest, false},

			// Web interface uses these.
			{api.SetDomainPath, setDomain, true},
			{api.DelDomainPath, delDomain, true},
			{api.GetDomainsByProjectPath, getDomainsByProject, true},

			// Other server stuff uses these.
			{api.GetUsersByKeyPath, getUsersByKey, true},
			{api.GetProjectAddrsByKeyPath, getProjectAddrsByKey, true},
			{api.GetProjectAddrByDomainPath, getProjectAddrByDomain, false},
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
		listenAddr := viper.GetString("listen")
		err = http.ListenAndServe(listenAddr, hzhttp.BaseContext(baseCtx, logMux))
		if err != nil {
			logger.Error("Couldn't serve on %v: %v", listenAddr, err)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.String("listen", ":8000", "HTTP listening address")

	pf.String("shared_secret",
		"/secrets/api-shared-secret/api-shared-secret",
		"Location of API shared secret")

	pf.String("token_secret",
		"/secrets/token-secret/token-secret",
		"Location of token secret file")

	pf.String("cluster_name", "horizon-cloud-1239",
		"Name of the GCE cluster to use.")

	pf.String("template_path",
		os.Getenv("GOPATH")+"/src/github.com/rethinkdb/horizon-cloud/templates/",
		"Path to the templates to use when creating Kube objects.")

	pf.String("storage_bucket_file",
		"",
		"File containing name of storage bucket to write user objects to")

	pf.String("service_account",
		"/secrets/gcloud-service-account/gcloud-service-account.json",
		"Path to the JSON service account.")

	pf.String("rethinkdb_addr", "localhost:28015",
		"Host and port of rethinkdb instance")

	pf.String("kube_namespace", "dev",
		"Kubernetes namespace to put pods in.")

	viper.BindPFlags(pf)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("hzc")
	viper.AutomaticEnv() // read in environment variables that match
}
