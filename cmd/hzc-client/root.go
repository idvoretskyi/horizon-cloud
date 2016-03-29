package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "hzc-client",
	Short: "Horizon Cloud Client",
	Long:  `A client for accessing Horizon Cloud.`,
}

// RSI: we need a domain name.
var server = "http://54.193.31.201:8000"

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringVarP(&cfgFile, "config", "c", ".hz/cloudconf.toml", "config file")
	pf.StringP("name", "n", "", "Project name (overrides config).")
	pf.StringP("identity_file", "i", "", "private key")
	pf.StringP("server", "S", server, "address of horizon cloud server")

	viper.BindPFlags(pf)
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