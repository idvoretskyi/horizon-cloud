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

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "hzc-client",
	Short: "Horizon Cloud Client",
	Long:  `A client for accessing Horizon Cloud.`,
}

func init() {
	const (
		apiServer   = "http://api.hzc.io"
		sshServer   = "ssh.hzc.io"
		fingerprint = "ssh.hzc.io ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCgSnlMGdhrthP2Dgjsp4lIg2Lzsy3ZdOYg0IucHHHJuiLDhKO9rIbg5GIHwJkTbV79ILMGNs+GmvRX2CLo7BbPeDqsdFETEpl0B8lMYz6/uvxZdTUDEBWQHXj3uYPsohxXAMgEQZqvNiE4UTBGsRc1aHYxxlcr3tPwJS76hs6wh9JEnPvU+p6AQ4CaJJzT/50EadgExrD7+I7UecJeB8IMD8+r1ChszzEcZlAcOIxLSVHpgWaR65XMPnSCl7WWRWyb17LDJQfwgq2SriAu83QiicdQE44CW10o2im4I4J/Vqs9nnWR4nlol9sRYBLkIxhJJ4ObI88Qt1yll32kwd/d"
		configFile  = ".hz/cloudconf.toml"
	)

	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.StringVarP(&cfgFile, "config", "c", configFile, "config file")
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
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
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
