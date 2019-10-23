package observer

import (
	"github.com/insolar/insolar/insolar"
)

type Notification struct {
	Ref            insolar.Reference
	GroupReference insolar.Reference
	UserReference  insolar.Reference
	Type           NotificationType
	Timestamp      int64
}

type NotificationStorage interface {
	Insert(Notification) error
}

type NotificationCollector interface {
	Collect(*Record) *Notification
}

// NotificationType type of swap steps
type NotificationType int

//go:generate stringer -type=NotificationType
const (
	NotificationInvite NotificationType = iota + 1
	NotificationContribution
	NotificationDeactivate
)
