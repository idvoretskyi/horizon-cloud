package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rethinkdb/horizon-cloud/internal/api"
	"github.com/rethinkdb/horizon-cloud/internal/ssh"
	"github.com/rethinkdb/horizon-cloud/internal/types"
	"github.com/rethinkdb/horizon-cloud/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(deployCmd)
}

func getToken() (string, error) {
	log.Printf("Getting deploy token...")

	sshServer := viper.GetString("ssh_server")
	identityFile := viper.GetString("identity_file")

	kh, err := ssh.NewKnownHosts([]string{viper.GetString("ssh_fingerprint")})
	if err != nil {
		return "", err
	}
	defer kh.Close()

	sshClient := ssh.New(ssh.Options{
		Host:         sshServer,
		User:         "auth",
		KnownHosts:   kh,
		IdentityFile: identityFile,
	})

	cmd := sshClient.Command("")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error %s: %s", err, buf.String())
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(buf.Bytes(), &errResp) == nil && errResp.Error != "" {
		return "", errors.New(errResp.Error)
	}

	var realResponse struct {
		Token string
	}
	err = json.Unmarshal(buf.Bytes(), &realResponse)
	if err != nil {
		return "", fmt.Errorf("couldn't unmarshal %#v: %v", buf.String(), err)
	}

	return realResponse.Token, nil
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy a project",
	Long:  `Deploy the specified project.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("Fetching deploy token...")
		token, err := getToken()
		if err != nil {
			fmt.Println("Couldn't get an API token: %v", err)
			fmt.Println()
			fmt.Println("Horizon Cloud is in private alpha. If you have been invited already,")
			fmt.Println("make sure your SSH key has been " +
				"added to your Horizon Cloud account.")
			os.Exit(1)
		}

		apiClient, err := api.NewClient(viper.GetString("api_server"), "")
		if err != nil {
			log.Fatalf("Couldn't create API client: %v", err)
		}

		name := viper.GetString("name")
		saveName := false
		if name == "" {
			saveName = true
			resp, err := apiClient.GetProjectsByToken(api.GetProjectsByTokenReq{token})
			if err != nil {
				log.Fatal(err)
			}
			if len(resp.Projects) == 0 {
				log.Fatalf("You aren't allowed to deploy to any projects.")
			}
			var newProjects []*types.Project
			var oldProjects []*types.Project
			for _, p := range resp.Projects {
				if p.HasBeenDeployedTo() {
					oldProjects = append(oldProjects, p)
				} else {
					newProjects = append(newProjects, p)
				}
			}
			fmt.Printf("No project name specified.  "+
				"Choose which project you wish to deploy to (it will be saved to %s):\n",
				configFile)

			width := len(fmt.Sprintf("%d", len(resp.Projects)))
			fmtSpec := fmt.Sprintf("%%%dd", width)
			fmt.Printf("%s\n", fmtSpec)
			offset := 1
			for _, p := range newProjects {
				fmt.Printf(fmtSpec+". NEW: %s\n", offset, p.SlashName())
				offset++
			}
			for _, p := range oldProjects {
				fmt.Printf(fmtSpec+". %s\n", offset, p.SlashName())
				offset++
			}

			var defaultProject *types.Project
			if len(newProjects) == 1 {
				defaultProject = newProjects[0]
			} else if len(newProjects) == 0 && len(oldProjects) == 1 {
				defaultProject = oldProjects[0]
			}

			if defaultProject != nil {
				fmt.Printf("(default 1) > ")
			} else {
				fmt.Printf("> ")
			}

			reader := bufio.NewReader(os.Stdin)
			name, err = reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if err != nil {
				log.Fatalf("error reading name: %v", err)
			}

			if name == "" && defaultProject != nil {
				name = defaultProject.SlashName()
			} else {
				if strings.Count(name, "/") == 0 {
					nameOffset := -1
					scanned, err := fmt.Sscanf(name, "%d", &nameOffset)
					nameOffset -= 1
					if scanned == len(name) && err == nil {
						if nameOffset >= 0 {
							if nameOffset < len(newProjects) {
								name = newProjects[nameOffset].SlashName()
							} else {
								nameOffset -= len(newProjects)
								if nameOffset < len(oldProjects) {
									name = oldProjects[nameOffset].SlashName()
								}
							}
						}
					}
				}
			}
		}

		log.Printf("Deploying %s...", name)

		log.Printf("Generating local file list...")

		files, err := createFileList("dist")
		if err != nil {
			log.Fatal(err)
		}

		schema, err := ioutil.ReadFile(schemaFile)
		if err != nil {
			log.Fatalf("unable to read schema at %v: %v", schemaFile, err)
		}

		triesLeft := 5
		for triesLeft > 0 {
			triesLeft--

			log.Printf("Checking local manifest against server...")

			nameParts := strings.Split(name, "/")
			userName := ""
			projectName := ""
			if len(nameParts) == 1 {
				projectName = nameParts[0]
			} else if len(nameParts) == 2 {
				userName = nameParts[0]
				projectName = nameParts[1]
			} else {
				log.Fatalf("invalid project name `%s` (has %d parts, needs 1 or 2)",
					name, len(nameParts))
			}
			resp, err := apiClient.UpdateProjectManifest(api.UpdateProjectManifestReq{
				ProjectID:     types.NewProjectID(userName, projectName),
				Files:         files,
				Token:         token,
				HorizonConfig: schema,
			})
			if err != nil {
				log.Fatal(err)
			}

			if len(resp.NeededRequests) == 0 {
				break
			}

			log.Printf("%v updates needed.", len(resp.NeededRequests))

			err = sendRequests("dist", resp.NeededRequests)
			if err != nil {
				log.Fatal(err)
			}
		}

		if triesLeft == 0 {
			log.Fatal("Couldn't deploy; too many retries. Maybe another deploy is running?")
		}

		if saveName {
			log.Printf("Saving name `%s` to `%s`.", name, configFile)

			f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				log.Fatal("error opening `%s` for appending: %v", configFile, err)
			}
			defer f.Close()
			_, err = f.WriteString(fmt.Sprintf("\nname = \"%s\"\n", name))
			if err != nil {
				log.Fatal("error writing to `%s`: %v", configFile, err)
			}
		}

		log.Printf("Deploy complete!\n")
	},
}

var skipNames = map[string]struct{}{
	"thumbs.db": struct{}{},
}

func createFileList(basePath string) ([]types.FileDescription, error) {
	files, err := walk(basePath)
	if err != nil {
		return nil, err
	}

	trim := basePath + string(filepath.Separator)
	for i := range files {
		files[i].Path = filepath.ToSlash(strings.TrimPrefix(files[i].Path, trim))
		if strings.HasPrefix(files[i].Path, "horizon/") {
			return nil, errors.New(
				"the directory name \"horizon\" is reserved and must not exist")
		}
	}

	return files, nil
}

func walk(basePath string) ([]types.FileDescription, error) {
	f, err := os.Open(basePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if !st.IsDir() {
		var desc types.FileDescription
		desc.Path = basePath

		var buf [16384]byte
		h := md5.New()
		for {
			n, err := f.Read(buf[:])
			_, _ = h.Write(buf[:n])
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			if desc.ContentType == "" {
				desc.ContentType = http.DetectContentType(buf[:n])
			}
		}

		desc.MD5 = h.Sum(nil)

		return []types.FileDescription{desc}, nil
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	sort.Strings(names)

	ret := make([]types.FileDescription, 0, 8)
	for _, name := range names {
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := skipNames[strings.ToLower(name)]; ok {
			continue
		}

		path := filepath.Join(basePath, name)

		inner, err := walk(path)
		if err != nil {
			return nil, err
		}

		ret = append(ret, inner...)
	}

	return ret, nil
}

func sendRequests(baseDir string, uploads []types.FileUploadRequest) error {
	for _, upload := range uploads {
		err := sendRequest(baseDir, upload)
		if err != nil {
			return err
		}
	}
	return nil
}

func sendRequest(baseDir string, upload types.FileUploadRequest) error {
	var body io.Reader
	if !util.IsSafeRelPath(upload.SourcePath) {
		return fmt.Errorf("%#v is not a safe path", upload.SourcePath)
	}
	if upload.SourcePath != "" {
		fh, err := os.Open(filepath.Join(baseDir, filepath.FromSlash(upload.SourcePath)))
		if err != nil {
			return err
		}
		defer fh.Close()
		body = fh
	}

	log.Printf("Uploading %v", upload.SourcePath)

	r, err := http.NewRequest(upload.Method, upload.URL, body)
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "")
	for k, v := range upload.Headers {
		r.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Got unexpected status code %v from %v %v: %#v", resp.StatusCode, upload.Method, upload.URL, string(body))
	}

	return nil
}
