package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "horizon-cloud-client",
	Short: "Horizon Cloud Client",
	Long:  `A client for accessing Horizon Cloud.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

// RSI: we need a domain name.
var server = "http://54.193.31.201:8000"

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringVarP(&cfgFile, "config", "c", ".hz/cloudconf.toml", "config file")

	pf.StringP("name", "n", "", "Project name (overrides config).")
	viper.BindPFlag("name", pf.Lookup("name"))

	pf.StringP("identity_file", "i", "", "private key")
	viper.BindPFlag("identity_file", pf.Lookup("identity_file"))

	pf.StringP("server", "S", server, "address of horizon cloud server")
	viper.BindPFlag("server", pf.Lookup("server"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigType("toml")

	viper.SetConfigName("cloudconf") // name of config file (without extension)
	viper.AddConfigPath(".hz")       // adding home directory as first search path
	// RSI: search parent directories?

	// RSI: do this?
	// viper.AddConfigPath("~/.hz") // adding home directory as first search path
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("No config file found (%s).  Continuing...\n", err)
	} else {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
