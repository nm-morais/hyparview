package protocol

import (
	"time"

	"github.com/nm-morais/go-babel/pkg/timer"
)

const ShuffleTimerID = 1501

type ShuffleTimer struct {
	duration time.Duration
}

func (ShuffleTimer) ID() timer.ID {
	return ShuffleTimerID
}

func (s ShuffleTimer) Duration() time.Duration {
	return s.duration
}

const PromoteTimerID = 1502

type PromoteTimer struct {
	duration time.Duration
}

func (PromoteTimer) ID() timer.ID {
	return PromoteTimerID
}

func (s PromoteTimer) Duration() time.Duration {
	return s.duration
}

const DebugTimerID = 1503

type DebugTimer struct {
	duration time.Duration
}

func (DebugTimer) ID() timer.ID {
	return DebugTimerID
}

func (s DebugTimer) Duration() time.Duration {
	return s.duration
}

const MaintenanceTimerID = 1504

type MaintenanceTimer struct {
	duration time.Duration
}

func (MaintenanceTimer) ID() timer.ID {
	return MaintenanceTimerID
}

func (s MaintenanceTimer) Duration() time.Duration {
	return s.duration
}
