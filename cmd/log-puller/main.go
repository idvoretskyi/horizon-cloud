package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/googleapi"
	"google.golang.org/cloud/pubsub"
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(-1)
	}
}

func writeToDB(session *r.Session, m *pubsub.Message) {
	var dbEntry struct {
		ID         string            `gorethink:"id"`
		Data       interface{}       `gorethink:"data,omitempty"`
		Attributes map[string]string `gorethink:"attributes,omitempty"`
	}
	dbEntry.ID = m.ID
	dbEntry.Attributes = m.Attributes

	if dbEntry.ID == "" {
		log.Printf("Got log entry with no ID")
		return
	}

	err := json.Unmarshal(m.Data, &dbEntry.Data)
	if err != nil {
		log.Printf("Couldn't decode log message as JSON: %v", err)
		return
	}

	table := r.DB(viper.GetString("dbname")).Table(viper.GetString("table"))
	res, err := table.Insert(dbEntry, r.InsertOpts{
		Durability: "soft",
		Conflict:   "replace",
	}).RunWrite(session)
	if err != nil {
		log.Fatal(err)
	}

	if res.Inserted+res.Replaced+res.Unchanged != 1 {
		log.Fatalf("Wrong number of changes in response %#v while inserting %#v", res, dbEntry)
	}

	return
}

func shouldKeepMessage(keepClusters map[string]struct{}, m *pubsub.Message) bool {
	namespace := m.Attributes["container.googleapis.com/namespace_name"]
	if namespace == "kube-system" {
		// kube-dns and heapster are INCREDIBLY TALKATIVE and they are
		// responsible for the vast majority of the logs. We skip all
		// the kube-system stuff since we have no control over it.
		return false
	}

	cluster := m.Attributes["container.googleapis.com/cluster_name"]
	if _, ok := keepClusters[cluster]; !ok {
		return false
	}

	return true
}

var RootCmd = &cobra.Command{
	Use: "log-puller",

	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(log.Lshortfile)

		keepClusters := make(map[string]struct{}, 2)
		for _, cluster := range strings.Split(viper.GetString("clusters"), ",") {
			if cluster == "" {
				continue
			}
			keepClusters[cluster] = struct{}{}
		}
		if len(keepClusters) == 0 {
			log.Fatal("At least one cluster must be specified")
		}

		session, err := r.Connect(r.ConnectOpts{
			Address: viper.GetString("rethinkdb_addr"),
		})
		if err != nil {
			log.Fatal(err)
		}

		ctx := context.Background()
		client, err := pubsub.NewClient(ctx, viper.GetString("project"))
		if err != nil {
			log.Fatal(err)
		}

		sub := client.Subscription(viper.GetString("subscription"))
		it, err := sub.Pull(ctx, pubsub.MaxPrefetch(1000))
		if err != nil {
			log.Fatal(err)
		}

		for {
			m, err := it.Next()
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 429 {
					// Too many requests. Wait a bit and try again.
					time.Sleep(time.Second * 10)
					continue
				}
				log.Fatal(err)
			}

			if shouldKeepMessage(keepClusters, m) {
				writeToDB(session, m)
			}

			m.Done(true)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	pf := RootCmd.PersistentFlags()

	pf.String("project", "horizon-cloud-1239",
		"Google Cloud project ID")
	pf.String("subscription", "hzc-logs-dev",
		"Pubsub subscription name")
	pf.String("rethinkdb_addr", "localhost:28015",
		"Host and port of rethinkdb instance")
	pf.String("dbname", "logs", "Database name")
	pf.String("table", "logs", "Table name")
	pf.String("clusters", "horizon-cloud,horizon-cloud-sys",
		"List of GKE cluster names to keep (comma separated)")

	viper.BindPFlags(pf)
}

func initConfig() {
	viper.SetEnvPrefix("log_puller")
	viper.AutomaticEnv()
}
