//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package dbconn

import (
	"github.com/go-pg/pg"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
)

func Connect(cfg configuration.DB) (*pg.DB, error) {
	opt, err := pg.ParseURL(cfg.URL)
	if err != nil {
		// pg.ParseURL uses standard url.Parse
		// witch fills url-string with password into error.
		// So we can't use errors.Wrap here and print error above in code.
		return nil, errors.New("failed to parse cfg.DB.URL")
	}
	opt.PoolSize = cfg.PoolSize
	return pg.Connect(opt), nil
}
