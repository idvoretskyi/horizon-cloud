package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
)

func autoFindName() string {
	data, err := ioutil.ReadFile("package.json")
	if err != nil {
		log.Fatal("No project name specified and `package.json` does not exist.")
	}
	var npmPackage struct {
		Name string `json:"name"`
	}
	if err = json.Unmarshal(data, &npmPackage); err != nil {
		log.Fatalf("No project name specified and failed to parse `package.json`: %s", err)
	}
	if npmPackage.Name == "" {
		log.Fatal("No project name specified and `package.json` does not include one.")
	}
	return npmPackage.Name
}

func missing(f string) bool {
	_, err := os.Stat(f)
	miss := os.IsNotExist(err)
	if err != nil && !miss {
		log.Fatalf("Error statting `%s`: `%s`", f, err)
	}
	return miss
}

func ensureDir(s string) {
	err := os.Mkdir(s, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Unable to create directory `%s`.", s)
	}
}

func ensureKey() {
	f1 := ".fusion/deploy_key"
	f2 := ".fusion/deploy_key.pub"
	privMissing := missing(f1)
	pubMissing := missing(f2)
	if privMissing != pubMissing {
		log.Fatalf("Only one of `%s` `%s` is present.", f1, f2)
	}
	if privMissing && pubMissing {
		cmd := exec.Command("ssh-keygen", "-q", "-f", f1, "-N", "")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Unable to create new key pair: %s", err)
		}
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	if len(os.Args) < 2 {
		log.Fatal("No subcommand specified.")
	}
	switch os.Args[1] {
	case "deploy":
		ensureDir(".fusion")
		var name string
		if len(os.Args) >= 3 {
			name = os.Args[2]
		} else {
			name = autoFindName()
		}
		spew.Dump(name)
		ensureKey()
	default:
		log.Fatalf("Unrecognized subcommand `%s`.", os.Args[1])
	}
}
