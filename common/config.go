package common

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
import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Config is representation of the configuration data
type Config struct {
	Account       Account
	AccountsMap   map[string]string
	KmsKeyTags    []string
	Flywheel      Flywheel
	ListenAddress string
	LogLevel      string
	Org           string
	Token         string
	Version       Version
}

// Account is the configuration for an individual account
type Account struct {
	Akid            string
	Endpoint        string
	ExternalID      string
	Region          string
	Role            string
	Secret          string
	DefaultSgs      []string
	DefaultSubnets  []string
	DefaultKmsKeyId string
}

// Flywheel is the configuration for task tracking in flywheel
type Flywheel struct {
	Namespace     string
	RedisAddress  string
	RedisDatabase string
	RedisUsername string
	RedisPassword string
	TTL           string
}

// Version carries around the API version information
type Version struct {
	Version           string
	VersionPrerelease string
	BuildStamp        string
	GitHash           string
}

// ReadConfig decodes the configuration from an io Reader
func ReadConfig(r io.Reader) (Config, error) {
	var c Config
	log.Infoln("Reading configuration")
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return c, errors.Wrap(err, "unable to decode JSON message")
	}
	return c, nil
}
