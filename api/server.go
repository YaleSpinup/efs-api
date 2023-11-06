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

package api

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/YaleSpinup/aws-go/services/session"
	"github.com/YaleSpinup/efs-api/common"
	"github.com/YaleSpinup/efs-api/ec2"
	"github.com/YaleSpinup/efs-api/efs"
	"github.com/YaleSpinup/efs-api/resourcegroupstaggingapi"
	"github.com/YaleSpinup/flywheel"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type server struct {
	accountsMap          map[string]string
	context              context.Context
	ec2Services          ec2.EC2
	efsServices          efs.EFS
	kmsKeyTags           []string
	flywheel             *flywheel.Manager
	org                  string
	orgPolicy            string
	rgTaggingAPIServices resourcegroupstaggingapi.ResourceGroupsTaggingAPI
	router               *mux.Router
	session              session.Session
	sessionCache         *cache.Cache
	version              common.Version
}

// NewServer creates a new server and starts it
func NewServer(config common.Config) error {
	// setup server context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if config.Org == "" {
		return errors.New("'org' cannot be empty in the configuration")
	}

	s := server{
		accountsMap:          config.AccountsMap,
		ec2Services:          ec2.EC2{},
		efsServices:          efs.EFS{},
		kmsKeyTags:           config.KmsKeyTags,
		rgTaggingAPIServices: resourcegroupstaggingapi.ResourceGroupsTaggingAPI{},
		router:               mux.NewRouter(),
		version:              config.Version,
		context:              ctx,
		org:                  config.Org,
		sessionCache:         cache.New(600*time.Second, 900*time.Second),
	}

	orgPolicy, err := orgTagAccessPolicy(config.Org)
	if err != nil {
		return err
	}
	s.orgPolicy = orgPolicy

	// Create a new session used for authentication and assuming cross account roles
	log.Debugf("Creating new session with key '%s' in region '%s'", config.Account.Akid, config.Account.Region)
	s.session = session.New(
		session.WithCredentials(config.Account.Akid, config.Account.Secret, ""),
		session.WithRegion(config.Account.Region),
		session.WithExternalID(config.Account.ExternalID),
		session.WithExternalRoleName(config.Account.Role),
	)

	manager, err := newFlywheelManager(config.Flywheel)
	if err != nil {
		return fmt.Errorf("failed to create new flywheel manager: %s", err)
	}
	s.flywheel = manager

	publicURLs := map[string]string{
		"/v1/efs/ping":    "public",
		"/v1/efs/version": "public",
		"/v1/efs/metrics": "public",
	}

	// load routes
	s.routes()

	if config.ListenAddress == "" {
		config.ListenAddress = ":8080"
	}

	handler := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, TokenMiddleware([]byte(config.Token), publicURLs, s.router)))
	srv := &http.Server{
		Handler:      handler,
		Addr:         config.ListenAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof("starting listener on %s", config.ListenAddress)
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func newFlywheelManager(config common.Flywheel) (*flywheel.Manager, error) {
	opts := []flywheel.ManagerOption{}

	if config.RedisAddress != "" {
		opts = append(opts, flywheel.WithRedisAddress(config.RedisAddress))
	}

	if config.RedisUsername != "" {
		opts = append(opts, flywheel.WithRedisAddress(config.RedisUsername))
	}

	if config.RedisPassword != "" {
		opts = append(opts, flywheel.WithRedisAddress(config.RedisPassword))
	}

	if config.RedisDatabase != "" {
		db, err := strconv.Atoi(config.RedisDatabase)
		if err != nil {
			return nil, err
		}
		opts = append(opts, flywheel.WithRedisDatabase(db))
	}

	if config.TTL != "" {
		ttl, err := time.ParseDuration(config.TTL)
		if err != nil {
			return nil, err
		}
		opts = append(opts, flywheel.WithTTL(ttl))
	}

	manager, err := flywheel.NewManager(config.Namespace, opts...)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

// LogWriter is an http.ResponseWriter
type LogWriter struct {
	http.ResponseWriter
}

// Write log message if http response writer returns an error
func (w LogWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if err != nil {
		log.Errorf("Write failed: %v", err)
	}
	return
}

type rollbackFunc func(ctx context.Context) error

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]rollbackFunc) {
	if t == nil {
		return
	}

	timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	done := make(chan string, 1)
	go func() {
		tasks := *t
		log.Errorf("executing rollback of %d tasks", len(tasks))
		for i := len(tasks) - 1; i >= 0; i-- {
			f := tasks[i]
			if funcerr := f(timeout); funcerr != nil {
				log.Errorf("rollback task error: %s, continuing rollback", funcerr)
			}
			log.Infof("executed rollback task %d of %d", len(tasks)-i, len(tasks))
		}
		done <- "success"
	}()

	// wait for a done context
	select {
	case <-timeout.Done():
		log.Error("timeout waiting for successful rollback")
	case <-done:
		log.Info("successfully rolled back")
	}
}

type stop struct {
	error
}

// retry is stolen from https://upgear.io/blog/simple-golang-retry-function/
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			log.Debugf("sleeping for %s", sleep.String())
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}

// if we have an entry for the account name, return the associated account number
func (s *server) mapAccountNumber(name string) string {
	if a, ok := s.accountsMap[name]; ok {
		return a
	}
	return name
}
