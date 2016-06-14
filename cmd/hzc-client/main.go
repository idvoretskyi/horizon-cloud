package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

var configFile string
var schemaFile string

var RootCmd = &cobra.Command{
	Use:   "hzc-client",
	Short: "Horizon Cloud Client",
	Long:  `A client for accessing Horizon Cloud.`,
}

func init() {
	const (
		apiServer         = "http://api.hzc.io"
		sshServer         = "ssh.hzc.io"
		fingerprint       = "ssh.hzc.io ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDDVA/PSlzkXtduo7GROpkqDK1HP2csW8LtA2yokVvScgVsNAJRec7H/LK+d3hv2ephNAVZuWnXFUJnLvHnDeXoWRLAfxE7+3h9jzjhn4X3vJImNtj4jKeTUEgMQ6TOPovWqer6IxPY5snQPsYPoegExY7LMRGnI2URQRXTHqru5jrox3mzEO4qJUNkC1YU5oNSAp+A/+sYNFofkL9t+eKL0P7qm+XOwGy3LbO5wbxnDo3e630ii/jYoKbLlSAcsP85ynHrQlydlQ1Onu4yPtqZaU5APeJMAWdkZ9kAkwTpyjgD67im6yeqCEa0Og2Nrjd0zMScbIUf359/cHwDmPUZ"
		defaultConfigFile = ".hz/cloudconf.toml"
		defaultSchemaFile = ".hz/schema.toml"
	)

	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringVarP(&configFile, "config", "c", defaultConfigFile, "config file")
	pf.StringVarP(&schemaFile, "schema", "k", defaultSchemaFile, "horizon schema file")

	pf.StringP("name", "n", "", "Project name (overrides config).")
	pf.StringP("identity_file", "i", "", "private key")

	pf.StringP("api_server", "s", apiServer, "horizon cloud API server base URL")
	pf.StringP("ssh_server", "S", sshServer, "address of horizon cloud ssh server")
	pf.StringP("ssh_fingerprint", "f", fingerprint,
		"fingerprint of horizon cloud ssh server")

	viper.BindPFlags(pf)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if configFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(configFile)
	}
	viper.SetConfigType("toml")

	viper.SetConfigName("cloudconf") // name of config file (without extension)
	viper.AddConfigPath(".hz")       // adding home directory as first search path
	// TODO: maybe do this, maybe search parent directories, whatever Horizon does.
	// viper.AddConfigPath("~/.hz") // adding home directory as first search path
	viper.SetEnvPrefix("hzc")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("No config file found (%s).  Continuing...\n", err)
	} else {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
