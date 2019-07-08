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

package internal

import (
	"context"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/log"
)

// Logger is a default insolar logger preset.
func Logger(
	ctx context.Context,
	cfg configuration.Log,
	traceID, nodeRef, nodeRole string,
) (context.Context, insolar.Logger) {
	inslog, err := log.NewLog(cfg)
	if err != nil {
		panic(err)
	}

	if newInslog, err := inslog.WithLevel(cfg.Level); err != nil {
		inslog.Error(err.Error())
	} else {
		inslog = newInslog
	}

	ctx = inslogger.SetLogger(ctx, inslog)
	ctx, _ = inslogger.WithTraceField(ctx, traceID)
	ctx, _ = inslogger.WithField(ctx, "nodeid", nodeRef)
	ctx, inslog = inslogger.WithField(ctx, "role", nodeRole)

	ctx = inslogger.SetLogger(ctx, inslog.WithField("loginstance", "inslog"))
	log.SetGlobalLogger(inslog.WithSkipFrameCount(1))

	return ctx, inslog.WithField("loginstance", "Logger")
}
