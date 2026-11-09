package main

import (
	"reflect"
	"testing"
	"time"
)

func TestGroupResultsAreIndependent(t *testing.T) {
	s := testServer()
	now := time.Now()
	s.enqueue("blind", "up", "manual")
	s.finishCommand(s.queue[0], "radio command failed", now)
	if len(s.groupStates()) != 0 {
		t.Fatal("individual action created group feedback")
	}
	s.queue = nil
	s.enqueue("all", "down", "manual")
	groupCommand := s.queue[0]
	s.queue = nil
	s.finishCommand(groupCommand, "Sent (position unknown)", now)
	before := s.groupStates()
	s.enqueue("blind", "stop", "manual")
	s.finishCommand(s.queue[0], "radio command failed", now.Add(time.Second))
	if !reflect.DeepEqual(s.groupStates(), before) {
		t.Fatal("individual result changed completed group feedback")
	}
	if before["all"].Sent != 1 || before["all"].Status != "Complete" {
		t.Fatal(before)
	}
}

func TestGroupPartialResultsAndSupersession(t *testing.T) {
	s := testServer()
	s.config.Devices = append(s.config.Devices, Device{ID: "second"}, Device{ID: "third"})
	s.config.Groups[0].Devices = []string{"blind", "second", "third"}
	now := time.Now()
	s.enqueue("all", "up", "manual")
	first, second := s.queue[0], s.queue[1]
	s.queue = s.queue[2:]
	s.finishCommand(first, "Sent (position unknown)", now)
	s.finishCommand(second, "command expired", now)
	if g := s.groupStates()["all"]; g.Pending != 1 || g.Sent != 1 || g.Failed != 1 {
		t.Fatal(g)
	}
	s.enqueue("third", "stop", "manual") // Replaces one waiting group command.
	g := s.groupStates()["all"]
	if g.Pending != 0 || g.Cancelled != 1 || g.Sent != 1 || g.Failed != 1 {
		t.Fatal(g)
	}
	s.enqueue("all", "down", "manual")
	s.finishCommand(first, "radio command failed", now.Add(time.Second))
	if g := s.groupStates()["all"]; g.Action != "down" || g.Pending != 3 || g.Failed != 0 {
		t.Fatal("old completion overwrote newer group request", g)
	}
	if d := s.states["blind"]; d.Action != "down" || d.Status != "Queued" {
		t.Fatal("old completion overwrote newer device request", d)
	}
}

func TestStoppedScheduleSettlesGroupResults(t *testing.T) {
	s := testServer()
	s.schedulerFile = t.TempDir() + "/scheduler.json"
	s.enqueue("all", "up", "schedule")
	if err := s.setScheduler(true, time.Now()); err != nil {
		t.Fatal(err)
	}
	g := s.groupStates()["all"]
	if g.Pending != 0 || g.Cancelled != 1 || g.Status != "Complete" {
		t.Fatal(g)
	}
}
