/*
Copyright © 2020 Yale University

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/YaleSpinup/efs-api/api"
	"github.com/YaleSpinup/efs-api/common"

	log "github.com/sirupsen/logrus"
)

var (
	// Version is the main version number
	Version = "0.0.0"

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = ""

	// Buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	Buildstamp = "No BuildStamp Provided"

	// Githash is the git sha of the built binary, it should be set at buildtime with ldflags
	Githash = "No Git Commit Provided"

	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("unable to get working directory")
	}
	log.Infof("Starting efs-api version %s%s (%s)", Version, VersionPrerelease, cwd)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("unable to read configuration from %s.  %+v", *configFileName, err)
	}

	config.Version = common.Version{
		Version:           Version,
		VersionPrerelease: VersionPrerelease,
		BuildStamp:        Buildstamp,
		GitHash:           Githash,
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if config.LogLevel == "debug" {
		log.Debug("starting profiler on 127.0.0.1:6080")
		go http.ListenAndServe("127.0.0.1:6080", nil)
	}
	log.Debugf("read config: %+v", config)

	if err := api.NewServer(config); err != nil {
		log.Fatal(err)
	}
}

func vers() {
	fmt.Printf("efs-api version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
