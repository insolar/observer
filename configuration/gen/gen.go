// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/configuration/api"
)

func main() {
	cfgs := make(map[string]interface{})
	cfgs["observerapi.yaml"] = api.Default()
	cfgs["observer.yaml"] = configuration.Default()
	cfgs["migrate.yaml"] = configuration.Migrate{}.Default()

	for filePath, cfg := range cfgs {
		out, _ := yaml.Marshal(cfg)
		err := ioutil.WriteFile(filePath, out, 0644)
		if err != nil {
			log := inslogger.FromContext(context.Background())
			log.Error(errors.Wrapf(err, "failed to write config file"))
			return
		}
		fmt.Println(filePath)
	}
}
