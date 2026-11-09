package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSchedulerStopResumeAndPersistence(t *testing.T) {
	s := testServer()
	s.schedulerFile = filepath.Join(t.TempDir(), "scheduler.json")
	s.config.Timezone = "UTC"
	s.config.Devices = append(s.config.Devices, Device{ID: "manual", Name: "Manual", RadioID: 2, House: 1235, Unit: 1})
	s.config.Schedules = []Schedule{{ID: "morning", Enabled: true, Target: "blind", Action: "up", Days: []int{1}, Event: "time", Time: "08:00"}}
	at := time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	s.enqueue("blind", "up", "schedule")
	s.enqueue("manual", "down", "manual")
	if err := s.setScheduler(true, at.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(s.queue) != 1 || s.queue[0].Source != "manual" {
		t.Fatal("stop removed manual command or retained scheduled command")
	}
	s.due(at.Add(-time.Second), at, map[string]bool{})
	if len(s.queue) != 1 {
		t.Fatal("stopped scheduler enqueued work")
	}
	restarted := testServer()
	restarted.schedulerFile = s.schedulerFile
	if err := restarted.loadScheduler(); err != nil || !restarted.schedulerStopped {
		t.Fatal("stop did not survive restart", err)
	}
	if err := s.setScheduler(false, at.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	s.due(at.Add(-time.Second), at.Add(3*time.Second), map[string]bool{})
	if len(s.queue) != 1 {
		t.Fatal("resuming replayed a missed event")
	}
	next := s.schedulerStatus(at.Add(3 * time.Second))
	if next.Next == nil || !next.Next.At.Equal(at.AddDate(0, 0, 7)) {
		t.Fatal("wrong next weekly event", next.Next)
	}
	if err := restarted.loadScheduler(); err != nil || restarted.schedulerStopped {
		t.Fatal("start did not persist", err)
	}
}
func TestSchedulerControlAuthorizationAndWriteFailure(t *testing.T) {
	s := testServer()
	s.schedulerFile = filepath.Join(t.TempDir(), "scheduler.json")
	if w := request(s, "POST", "/api/scheduler", `{"action":"stop"}`, map[string]string{"Content-Type": "application/json"}); w.Code != 401 {
		t.Fatal(w.Code)
	}
	tok := strings.Repeat("a", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	headers := map[string]string{"Cookie": cookieName + "=" + tok, "Content-Type": "application/json"}
	if w := request(s, "POST", "/api/scheduler", `{"action":"stop"}`, headers); w.Code != 403 {
		t.Fatal(w.Code)
	}
	headers["Authorization"] = "Bearer csrf"
	for _, tc := range []struct {
		method, body string
		want         int
	}{{"GET", "", 405}, {"POST", `{"action":"bad"}`, 400}, {"POST", `{"action":"stop"}`, 200}, {"POST", `{"action":"start"}`, 200}} {
		if w := request(s, tc.method, "/api/scheduler", tc.body, headers); w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	s.schedulerFile = t.TempDir() // A directory cannot be replaced by the state file.
	if err := s.setScheduler(true, time.Now()); err == nil || s.schedulerStopped {
		t.Fatal("failed save changed running state")
	}
}
