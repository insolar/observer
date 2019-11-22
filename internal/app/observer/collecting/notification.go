/*
 *
 *  Copyright  2019. Insolar Technologies GmbH
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package collecting

import (
	"fmt"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/log"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"reflect"
)

type NotificationCollector struct {
	log          *logrus.Logger
	collector    *ResultCollector
	user         *ChainCollector
	notification *ChainCollector
	activate     *ActivateCollector
}

func NewNotificationCollector(log *logrus.Logger) *NotificationCollector {
	collector := NewResultCollector(isNotificationCreationCall, successResult)
	activate := NewActivateCollector(isNotificationNew, isNotificationActivate)

	user := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isNotificationCreationCall(res.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return insolar.ID{}
			}
			return res.Request.ID
		},
		Proper: func(chain interface{}) bool {
			res, ok := chain.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isNotificationCreationCall(res.Request)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return request.IsIncoming()
		},
		Origin: func(chain interface{}) insolar.ID {
			request := observer.CastToRequest(chain)
			return request.Reason()
		},
		Proper: func(chain interface{}) bool {
			request := observer.CastToRequest(chain)
			return isCreateNotification(request)
		},
	})
	notification := NewChainCollector(&RelationDesc{
		Is: func(chain interface{}) bool {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			res, ok := che.Parent.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isNotificationCreationCall(res.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return insolar.ID{}
			}
			rec, ok := che.Child.(*observer.Record)
			if !ok {
				return insolar.ID{}
			}
			return rec.ID
		},
		Proper: func(chain interface{}) bool {
			che, ok := chain.(*observer.Chain)
			if !ok {
				return false
			}
			res, ok := che.Parent.(*observer.CoupledResult)
			if !ok {
				return false
			}
			return isNotificationCreationCall(res.Request)
		},
	}, &RelationDesc{
		Is: func(chain interface{}) bool {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return false
			}
			return isNotificationNew(act.Request)
		},
		Origin: func(chain interface{}) insolar.ID {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return insolar.ID{}
			}
			return act.Request.Reason()
		},
		Proper: func(chain interface{}) bool {
			act, ok := chain.(*observer.CoupledActivate)
			if !ok {
				return false
			}
			return isNotificationNew(act.Request)
		},
	})
	return &NotificationCollector{
		collector:    collector,
		user:         user,
		notification: notification,
		activate:     activate,
	}
}

type Notification struct {
	GroupRef         insolar.Reference
	MemberRef        insolar.Reference
	TypeNotification observer.NotificationType
}

func (c *NotificationCollector) Collect(rec *observer.Record) *observer.Notification {
	if rec == nil {
		return nil
	}
	res := c.collector.Collect(rec)
	act := c.activate.Collect(rec)
	var half *observer.Chain
	if isCreateNotification(rec) {
		half = c.user.Collect(rec)
	}
	if res != nil {
		half = c.user.Collect(res)
	}
	var chain *observer.Chain
	if half != nil {
		chain = c.notification.Collect(half)
	}
	if act != nil {
		chain = c.notification.Collect(act)
	}

	if chain == nil {
		return nil
	}

	coupleAct := c.unwrapNotificationChain(chain)

	m, err := c.build(coupleAct)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build notification"))
		return nil
	}
	return m
}

func (c *NotificationCollector) unwrapNotificationChain(chain *observer.Chain) *observer.Activate {
	log := c.log

	coupledAct, ok := chain.Child.(*observer.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.CoupledActivate", reflect.TypeOf(chain.Child)))
		return nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil
	}

	return coupledAct.Activate
}

func (c *NotificationCollector) build(act *observer.Activate) (*observer.Notification, error) {
	if act == nil {
		return nil, errors.New("trying to create notification from non complete builder")
	}

	var notification Notification

	err := insolar.Deserialize(act.Virtual.GetActivate().Memory, &notification)
	if err != nil {
		return nil, err
	}

	date, err := act.ID.Pulse().AsApproximateTime()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert notification create pulse (%d) to time", act.ID.Pulse())
	}

	fmt.Println("Insert new notification ref:", insolar.NewReference(act.ObjectID).String())
	return &observer.Notification{
		Ref:            *insolar.NewReference(act.ObjectID),
		UserReference:  notification.MemberRef,
		GroupReference: notification.GroupRef,
		Timestamp:      date.Unix(),
		Type:           notification.TypeNotification,
	}, nil
}

func isNotificationCreationCall(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	if !request.IsMemberCall() {
		return false
	}
	args := request.ParseMemberCallArguments()
	return args.Params.CallSite == "group.setNotification"
}

func isNotificationActivate(chain interface{}) bool {
	activate := observer.CastToActivate(chain)
	if !activate.IsActivate() {
		return false
	}
	act := activate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A5pYbHstfXoD4bf2iZh128mQmuS6BFxUfKqjTg5q")
	return act.Image.Equal(*prototypeRef)
}

func isNotificationNew(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "New" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A5pYbHstfXoD4bf2iZh128mQmuS6BFxUfKqjTg5q") // ntf
	return in.Prototype.Equal(*prototypeRef)
}

func isCreateNotification(chain interface{}) bool {
	request := observer.CastToRequest(chain)
	if !request.IsIncoming() {
		return false
	}

	in := request.Virtual.GetIncomingRequest()
	if in.Method != "CreateNotification" {
		return false
	}

	if in.Prototype == nil {
		return false
	}

	prototypeRef, _ := insolar.NewReferenceFromString("0111A5tDgkPiUrCANU8NTa73b7w6pWGRAUxJTYFXwTnR") // user
	return in.Prototype.Equal(*prototypeRef)
}
