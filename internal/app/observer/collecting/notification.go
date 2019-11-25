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
)

type NotificationCollector struct {
	log *logrus.Logger
}

func NewNotificationCollector(log *logrus.Logger) *NotificationCollector {
	return &NotificationCollector{
		log: log,
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
	actCandidate := observer.CastToActivate(rec)

	if !actCandidate.IsActivate() {
		return nil
	}

	act := actCandidate.Virtual.GetActivate()

	// TODO: import from platform
	prototypeRef, _ := insolar.NewReferenceFromString("0111A5pYbHstfXoD4bf2iZh128mQmuS6BFxUfKqjTg5q")
	if !act.Image.Equal(*prototypeRef) {
		return nil
	}

	n, err := c.build(actCandidate)
	if err != nil {
		log.Error(errors.Wrapf(err, "failed to build notification"))
		return nil
	}
	return n
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
