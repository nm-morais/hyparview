package protocol

import (
	"time"

	"github.com/nm-morais/go-babel/pkg/timer"
)

const ShuffleTimerID = 2001

type ShuffleTimer struct {
	duration time.Duration
}

func (ShuffleTimer) ID() timer.ID {
	return ShuffleTimerID
}

func (s ShuffleTimer) Duration() time.Duration {
	return s.duration
}

const PromoteTimerID = 2002

type PromoteTimer struct {
	duration time.Duration
}

func (PromoteTimer) ID() timer.ID {
	return PromoteTimerID
}

func (s PromoteTimer) Duration() time.Duration {
	return s.duration
}

const DebugTimerID = 2003

type DebugTimer struct {
	duration time.Duration
}

func (DebugTimer) ID() timer.ID {
	return DebugTimerID
}

func (s DebugTimer) Duration() time.Duration {
	return s.duration
}
