// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package registry

import (
	"strings"

	"github.com/pkg/errors"
)

type tagsResponse struct {
	Tags []string `json:"tags"`
}

// Tags returns the list of tags fo the repository.
func (registry *Registry) Tags(repository string) (tags []string, err error) {
	url := registry.url("/v2/%s/tags/list", repository)

	var response tagsResponse
	for {
		registry.Logf("registry.tags url=%s repository=%s", url, repository)
		url, err = registry.getPaginatedJson(url, &response)
		switch err {
		case ErrNoMorePages:
			tags = append(tags, response.Tags...)
			return tags, nil
		case nil:
			tags = append(tags, response.Tags...)
			continue
		default:
			if strings.Contains(err.Error(), "404") {
				return nil, errors.Wrap(ErrRepositoryNotFound, "list tags")
			}
			return nil, err
		}
	}
}
