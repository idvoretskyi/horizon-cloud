package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type config struct {
	APIClient *api.Client
}

var cfgFile string

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "hzc-http",
	Short: "hzc-http",
	Long:  `hzc-http`,
	Run: func(cmd *cobra.Command, args []string) {
		conf := &config{}

		log.SetFlags(log.Lshortfile)
		logger, err := hzlog.MainLogger("hzc-http")
		if err != nil {
			log.Fatal(err)
		}

		writerLogger := hzlog.WriterLogger(logger)
		log.SetOutput(writerLogger)

		baseCtx := hzhttp.NewContext(logger)

		secretPath := viper.GetString("secret_path")
		secret, err := ioutil.ReadFile(secretPath)
		if err != nil {
			log.Fatalf("Couldn't read api server secret from %v: %v", secretPath, err)
		}

		conf.APIClient, err = api.NewClient(viper.GetString("api_server"), string(secret))
		if err != nil {
			log.Fatalf("Couldn't create API client: %v", err)
		}

		var handler hzhttp.Handler = NewHandler(conf)
		handler = hzhttp.LogHTTPRequests(handler)

		plainHandler := hzhttp.BaseContext(baseCtx, handler)

		if tlsAddr := viper.GetString("listen_tls"); tlsAddr != "" {
			go func() {
				certFile := viper.GetString("tls_cert")
				keyFile := viper.GetString("tls_key")
				log.Fatal(http.ListenAndServeTLS(tlsAddr, certFile, keyFile, plainHandler))
			}()
		}

		log.Fatal(http.ListenAndServe(viper.GetString("listen"), plainHandler))
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringP("listen", "l", ":80", "Address to listen for HTTP connections on.")
	pf.StringP("listen_tls", "", "", "Address to listen for HTTPS connections on.")
	pf.StringP("tls_cert", "", "", "Path to TLS certificate chain")
	pf.StringP("tls_key", "", "", "Path to TLS key")
	pf.StringP("api_server", "a", "http://api-server:8000", "API server base URL.")
	pf.StringP("secret_path", "s",
		"/secrets/api-shared-secret/api-shared-secret",
		"Path to API server shared secret")

	viper.BindPFlags(pf)
}

// initConfig reads in ENV variables
func initConfig() {
	viper.AutomaticEnv()
}
