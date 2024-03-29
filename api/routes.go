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
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	api := s.router.PathPrefix("/v1/efs").Subrouter()

	api.HandleFunc("/ping", s.PingHandler).Methods(http.MethodGet)
	api.HandleFunc("/version", s.VersionHandler).Methods(http.MethodGet)
	api.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	api.Handle("/flywheel", s.flywheel.Handler())

	api.HandleFunc("/{account}/filesystems", s.FileSystemListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}", s.FileSystemListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}", s.FileSystemCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/filesystems/{group}/{id}", s.FileSystemShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}/{id}", s.FileSystemDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/filesystems/{group}/{id}", s.FileSystemUpdateHandler).Methods(http.MethodPut)

	api.HandleFunc("/{account}/filesystems/{group}/{id}/users", s.UsersCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/users", s.UsersListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/users/{user}", s.UsersShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/users/{user}", s.UsersUpdateHandler).Methods(http.MethodPut)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/users/{user}", s.UsersDeleteHandler).Methods(http.MethodDelete)

	api.HandleFunc("/{account}/filesystems/{group}/{id}/aps", s.FileSystemAPListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/aps", s.FileSystemAPCreateHandler).Methods(http.MethodPost)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/aps/{apid}", s.FileSystemAPShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/filesystems/{group}/{id}/aps/{apid}", s.FileSystemAPDeleteHandler).Methods(http.MethodDelete)
}
