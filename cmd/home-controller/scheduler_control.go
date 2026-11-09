package main

import (
	"errors"
	"os"
	"time"
)

type SchedulerPreference struct {
	Stopped bool `json:"stopped"`
}
type NextAction struct {
	At     time.Time `json:"at"`
	Target string    `json:"target"`
	Action string    `json:"action"`
}
type SchedulerStatus struct {
	Running  bool        `json:"running"`
	Enabled  int         `json:"enabled_schedules"`
	Timezone string      `json:"timezone"`
	Next     *NextAction `json:"next,omitempty"`
}

func (s *Server) loadScheduler() error {
	var p SchedulerPreference
	err := readJSON(s.schedulerFile, &p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	s.schedulerStopped = p.Stopped
	return nil
}

// Caller holds the server lock. A command already taken by the worker is in flight.
func (s *Server) setScheduler(stopped bool, now time.Time) error {
	if s.schedulerStopped == stopped {
		return nil
	}
	if err := atomicJSON(s.schedulerFile, SchedulerPreference{Stopped: stopped}); err != nil {
		return err
	}
	s.schedulerStopped = stopped
	s.previewAt = time.Time{}
	if stopped {
		pending := s.queue[:0]
		for _, cmd := range s.queue {
			if cmd.Source == "schedule" {
				s.finishCommand(cmd, "Schedule cancelled", now)
			} else {
				pending = append(pending, cmd)
			}
		}
		s.queue = pending
	} else {
		s.resumeAt = now
	}
	return nil
}
func (s *Server) schedulerStatus(now time.Time) SchedulerStatus {
	out := SchedulerStatus{Running: !s.schedulerStopped, Timezone: s.config.Timezone}
	for _, rule := range s.config.Schedules {
		if rule.Enabled {
			out.Enabled++
		}
	}
	if s.schedulerStopped || out.Enabled == 0 || now.Year() < 2025 {
		return out
	}
	if !s.previewAt.IsZero() && now.Sub(s.previewAt) >= 0 && now.Sub(s.previewAt) < time.Minute && (s.preview == nil || s.preview.At.After(now)) {
		out.Next = s.preview
		return out
	}
	loc, err := time.LoadLocation(s.config.Timezone)
	if err != nil {
		return out
	}
	today := now.In(loc)
	var next *NextAction
	for _, rule := range s.config.Schedules {
		if !rule.Enabled {
			continue
		}
		for delta := -1; delta <= 8; delta++ {
			day := time.Date(today.Year(), today.Month(), today.Day()+delta, 0, 0, 0, 0, loc)
			match := false
			for _, weekday := range rule.Days {
				if weekday == int(day.Weekday()) {
					match = true
				}
			}
			if !match {
				continue
			}
			at, err := s.eventTime(rule, day)
			if err != nil || !at.After(now) {
				continue
			}
			if next == nil || at.Before(next.At) {
				name := rule.Target
				for _, d := range s.config.Devices {
					if d.ID == rule.Target {
						name = d.Name
					}
				}
				for _, g := range s.config.Groups {
					if g.ID == rule.Target {
						name = g.Name
					}
				}
				next = &NextAction{at, name, rule.Action}
			}
			break
		}
	}
	s.previewAt, s.preview = now, next
	out.Next = next
	return out
}
func (s *Server) writeStatus(now time.Time) {
	atomicJSON(run+"/status.json", map[string]any{"devices": s.states, "group_states": s.groupStates(), "history": s.history, "queue": len(s.queue), "scheduler": s.schedulerStatus(now)})
}
