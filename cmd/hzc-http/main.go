package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/hzhttp"
	"github.com/rethinkdb/horizon-cloud/internal/hzlog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

type config struct {
	APIClient *api.Client
}

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringP("listen", "l", ":80", "Address to listen for HTTP connections on.")
	pf.StringP("api_server", "a", "http://api-server:8000", "API server base URL.")

	viper.BindPFlags(pf)
}

// initConfig reads in ENV variables
func initConfig() {
	viper.AutomaticEnv()
}

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

		conf.APIClient, err = api.NewClient(viper.GetString("api_server"), "")
		if err != nil {
			log.Fatalf("Couldn't create API client: %v", err)
		}

		var handler hzhttp.Handler = NewHandler(conf, baseCtx)
		handler = hzhttp.LogHTTPRequests(handler)

		plainHandler := hzhttp.BaseContext(baseCtx, handler)

		log.Fatal(http.ListenAndServe(viper.GetString("listen"), plainHandler))
	},
}
