// Copyright 2018 Paul Greenberg (greenpau@outlook.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"github.com/greenpau/go-ansible-db/pkg/db"
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	appName        = "go-ansible-db-client"
	appVersion     = "[untracked]"
	appDocs        = "https://github.com/greenpau/go-ansible-db/"
	appDescription = "Ansible DB (Inventory and Vault) client"
	gitBranch      string
	gitCommit      string
	buildUser      string // whoami
	buildDate      string // date -u
)

func main() {
	var logLevel string
	var isShowVersion bool

	var inputInventoryFile string
	var inputVaultFile string
	var inputVaultPassword string
	var inputVaultPasswordFile string

	flag.StringVar(&inputInventoryFile, "inventory", "hosts", "ansible inventory file")
	flag.StringVar(&inputVaultFile, "vault", "", "ansible vault file")
	flag.StringVar(&inputVaultPassword, "vault.key", "", "ansible vault password")
	flag.StringVar(&inputVaultPasswordFile, "vault.key.file", "", "ansible vault password file")
	flag.StringVar(&logLevel, "log.level", "info", "logging severity level")
	flag.BoolVar(&isShowVersion, "version", false, "version information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s - %s\n\n", appName, appDescription)
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n\n", appName)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDocumentation: %s\n\n", appDocs)
	}
	flag.Parse()
	if isShowVersion {
		fmt.Fprintf(os.Stdout, "%s %s", appName, appVersion)
		if gitBranch != "" {
			fmt.Fprintf(os.Stdout, ", branch: %s", gitBranch)
		}
		if gitCommit != "" {
			fmt.Fprintf(os.Stdout, ", commit: %s", gitCommit)
		}
		if buildUser != "" && buildDate != "" {
			fmt.Fprintf(os.Stdout, ", build on %s by %s", buildDate, buildUser)
		}
		fmt.Fprint(os.Stdout, "\n")
		os.Exit(0)
	}
	if level, err := log.ParseLevel(logLevel); err == nil {
		log.SetLevel(level)
	} else {
		log.Errorf(err.Error())
		os.Exit(1)
	}

	inv := db.NewInventory()
	if err := inv.LoadFromFile(inputInventoryFile); err != nil {
		log.Fatalf("argument '-inventory %s': %s", inputInventoryFile, err)
	}
	log.Debugf("inventory file: %s", inputInventoryFile)
	hosts, err := inv.GetHosts()
	if err != nil {
		log.Fatalf("GetHosts() failed: %s", err)
	}
	for _, h := range hosts {
		fmt.Fprintf(os.Stdout, "%s", h.Name)
	}
}
