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

package collecting

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/pkg/panic"
	"github.com/sirupsen/logrus"
)

type GroupUpdateCollector struct {
	log *logrus.Logger
}

func NewGroupUpdateCollector(log *logrus.Logger) *GroupUpdateCollector {
	return &GroupUpdateCollector{
		log: log,
	}
}

func (c *GroupUpdateCollector) Collect(rec *observer.Record) *observer.GroupUpdate {
	defer panic.Catch("group_update_collector")

	if rec == nil {
		return nil
	}

	v, ok := rec.Virtual.Union.(*record.Virtual_Amend)
	if !ok {
		return nil
	}
	if !isGroupAmend(v.Amend) {
		return nil
	}

	amd := rec.Virtual.GetAmend()
	group, err := groupUpdate(rec)

	if err != nil {
		logrus.Info(err.Error())
		return nil
	}

	return &observer.GroupUpdate{
		PrevState:   amd.PrevState,
		GroupState:  rec.ID,
		Goal:        group.Goal,
		Purpose:     group.Purpose,
		ProductType: group.ProductType,
	}
}

func isGroupAmend(amd *record.Amend) bool {
	prototypeRef, _ := insolar.NewReferenceFromBase58("0111A7bz1ZzDD9CJwckb5ufdarH7KtCwSSg2uVME3LN9")
	return amd.Image.Equal(*prototypeRef)
}
