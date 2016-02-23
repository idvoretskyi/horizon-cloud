package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "fusion-ops-client",
	Short: "Fusion Ops Client",
	Long:  `A client for accessing Fusion Ops.`,
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
var server = "http://localhost:8000"

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(
		&cfgFile, "config", "c", ".fusion/conf", "config file")

	RootCmd.PersistentFlags().StringP(
		"identity_file", "i", "~/.ssh/id_rsa", "private key")
	viper.BindPFlag("identity_file", RootCmd.PersistentFlags().Lookup("identity_file"))

	// RSI: should we instead be generating this from the private key?
	// We'd need to have a canonicalization step probably.
	RootCmd.PersistentFlags().StringP(
		"public_key_file", "k", "~/.ssh/id_rsa.pub", "public key")
	viper.BindPFlag("public_key_file", RootCmd.PersistentFlags().Lookup("public_key_file"))

	RootCmd.PersistentFlags().StringP(
		"server", "S", server, "address of fusion ops server")
	viper.BindPFlag("server", RootCmd.PersistentFlags().Lookup("server"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	viper.SetConfigType("yaml")

	viper.SetConfigName("conf")      // name of config file (without extension)
	viper.AddConfigPath(".fusion")   // adding home directory as first search path
	viper.AddConfigPath("~/.fusion") // adding home directory as first search path
	viper.AutomaticEnv()             // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("No config file found (%s).  Continuing...\n", err)
	} else {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
