package cmd

import (
	"log"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/ssh"
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
			log.Fatal("no project name specified (use `-n` or `.horizon/conf`)")
		}

		err = withSSHConnection(
			&commandContext{client, name, viper.GetString("identity_file")},
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
}
