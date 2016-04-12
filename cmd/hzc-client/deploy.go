package main

import (
	"log"

	"github.com/pborman/uuid"
	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy a project",
	Long:  `Deploy the specified project.`,
	Run: func(cmd *cobra.Command, args []string) {
		server := viper.GetString("server")
		identityFile := viper.GetString("identity_file")

		name := viper.GetString("name")
		if name == "" {
			log.Fatalf("no project name specified (use `-n` or `%s`)", configFile)
		}

		kh, err := ssh.NewKnownHosts([]string{viper.GetString("fingerprint")})
		if err != nil {
			log.Fatalf("failed to deploy: %s", err)
		}
		defer kh.Close()

		sshClient := ssh.New(ssh.Options{
			Host:         server,
			User:         "horizon",
			Environment:  map[string]string{api.ProjectEnvVarName: name},
			KnownHosts:   kh,
			IdentityFile: identityFile,
		})
		log.Printf("deploying to %s...", server)
		// RSI: check whether dist exists.
		dirName := uuid.New()
		err = sshClient.RsyncTo("dist/", "/data/"+dirName+"/", "/data/current/")
		if err != nil {
			log.Fatalf("failed to deploy: %s", err)
		}
		shellCmd := "DIR=" + ssh.ShellEscape(dirName) + " " + "/home/horizon/post-deploy.sh"
		err = sshClient.RunCommand("bash -c " + ssh.ShellEscape(shellCmd))
		if err != nil {
			log.Fatalf("failed to deploy: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(deployCmd)
}
