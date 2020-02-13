// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package main

import (
	"context"
	"io/ioutil"

	"github.com/insolar/insolar/instrumentation/inslogger"

	apiconfiguration "github.com/insolar/observer/configuration/api"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
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
