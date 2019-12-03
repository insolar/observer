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

package main

import (
	"context"
	"io/ioutil"

	"github.com/insolar/insolar/instrumentation/inslogger"

	apiconfiguration "github.com/insolar/observer/configuration/api"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func main() {
	cfg := apiconfiguration.Default()
	out, _ := yaml.Marshal(cfg)
	err := ioutil.WriteFile(apiconfiguration.ConfigFilePath, out, 0644)
	if err != nil {
		log := inslogger.FromContext(context.Background())
		log.Error(errors.Wrapf(err, "failed to write config file"))
		return
	}
}
