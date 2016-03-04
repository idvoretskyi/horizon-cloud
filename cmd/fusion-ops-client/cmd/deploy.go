package cmd

import (
	"io/ioutil"
	"log"

	"github.com/rethinkdb/fusion-ops/internal/api"
	"github.com/rethinkdb/fusion-ops/internal/ssh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy a project",
	Long:  `Deploy the specified project.  If the project doesn't exist, create it.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := api.NewClient(viper.GetString("server"))
		if err != nil {
			log.Fatalf("unable to create client: %s", err)
		}

		name := viper.GetString("name")
		if name == "" {
			log.Fatal("no project name specified (use `-n` or `.fusion/conf`)")
		}
		pubKey, err := ioutil.ReadFile(viper.GetString("public_key_file"))
		if err != nil {
			log.Fatalf("unable to read public key: %s", err)
		}

		err = withSSHConnection(
			&commandContext{client, name, string(pubKey), viper.GetString("identity_file")},
			api.AllowClusterStart,
			func(sshClient *ssh.Client, resp *api.WaitConfigAppliedResp) error {
				log.Printf("deploying to %v (%v)...", resp.Config, resp.Target)
				return nil
			},
		)
		if err != nil {
			log.Fatalf("failed to deploy: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(deployCmd)
	deployCmd.PersistentFlags().StringP(
		"name", "n", "", "Project name (overrides config).")
	viper.BindPFlag("name", deployCmd.PersistentFlags().Lookup("name"))
}
