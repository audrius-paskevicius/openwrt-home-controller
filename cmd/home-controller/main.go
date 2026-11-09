// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
	"unicode/utf8"
)

const base = "/etc/home-controller"
const run = "/var/run/home-controller"
const cookieName = "__Host-homecontroller"

var idRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
var errNoSolarEvent = errors.New("solar event does not occur")

type Device struct {
	Type      string `json:"type,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	HouseCode string `json:"house_code,omitempty"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	RadioID   int    `json:"radio_id"`
	House     int    `json:"house"`
	Unit      int    `json:"unit"`
	Icon      string `json:"icon,omitempty"`
}
type Group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Devices []string `json:"devices"`
	Icon    string   `json:"icon,omitempty"`
}
type Schedule struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	Target   string `json:"target"`
	Action   string `json:"action"`
	Days     []int  `json:"days"`
	Event    string `json:"event"`
	Time     string `json:"time"`
	Offset   int    `json:"offset"`
	Earliest string `json:"earliest"`
	Latest   string `json:"latest"`
	Fallback string `json:"fallback"`
}
type Config struct {
	Enabled             bool       `json:"enabled"`
	Port                int        `json:"port"`
	Networks            []string   `json:"networks"`
	Timezone            string     `json:"timezone"`
	Latitude            float64    `json:"latitude"`
	Longitude           float64    `json:"longitude"`
	Controller          string     `json:"controller,omitempty"` // Accepted only to migrate older configuration.
	PreferredController string     `json:"preferred_controller,omitempty"`
	SessionDays         int        `json:"session_days"`
	Devices             []Device   `json:"devices"`
	Groups              []Group    `json:"groups"`
	Schedules           []Schedule `json:"schedules"`
	SchedulerIcon       string     `json:"scheduler_icon,omitempty"`
	ControlOrder        []string   `json:"control_order,omitempty"`
}
type Credential struct {
	User     string `json:"user"`
	Salt     string `json:"salt"`
	Hash     string `json:"hash"`
	Revision string `json:"revision"`
}
type Session struct {
	Hash     string `json:"hash"`
	CSRF     string `json:"csrf"`
	Expires  int64  `json:"expires"`
	Revision string `json:"revision"`
	Created  int64  `json:"created"`
	Device   string `json:"device"`
}
type Command struct {
	Device  string
	Action  string
	Source  string
	Expires time.Time
	Target  string
	Request uint64
}
type DeviceState struct {
	Action string `json:"action"`
	Status string `json:"status"`
	At     int64  `json:"at"`
}
type Event struct {
	At     int64  `json:"at"`
	Device string `json:"device"`
	Action string `json:"action"`
	Source string `json:"source"`
	Result string `json:"result"`
}
type Server struct {
	mu               sync.Mutex
	config           Config
	cred             Credential
	sessions         []Session
	states           map[string]DeviceState
	history          []Event
	queue            []Command
	requests         map[string]*controlRequest
	latestDevice     map[string]uint64
	nextRequest      uint64
	wake             chan bool
	attempts         map[string][]time.Time
	sender           func(Command) error
	solar            func(Schedule, time.Time) (time.Time, error)
	schedulerFile    string
	schedulerStopped bool
	resumeAt         time.Time
	previewAt        time.Time
	preview          *NextAction
}

func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func atomicJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".new-")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	f.Close()
	if e != nil {
		return e
	}
	return os.Rename(n, path)
}
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// PBKDF2-HMAC-SHA256, one 32-byte block. Stored salt is 256 random bits.
func passwordHash(password, salt string) string {
	m := hmac.New(sha256.New, []byte(password))
	m.Write([]byte(salt))
	m.Write([]byte{0, 0, 0, 1})
	u := m.Sum(nil)
	out := append([]byte{}, u...)
	for i := 1; i < 210000; i++ {
		m.Reset()
		m.Write(u)
		u = m.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return hex.EncodeToString(out)
}

func validateSharedAccount(user, password string) error {
	if !idRE.MatchString(user) {
		return errors.New("username must start with a lowercase letter and contain only lowercase letters, numbers, hyphens or underscores (up to 32 characters)")
	}
	if password == "" {
		return errors.New("enter a new password")
	}
	if utf8.RuneCountInString(password) > 256 {
		return errors.New("password must be at most 256 characters")
	}
	return nil
}
func validAction(s string) bool {
	return s == "up" || s == "stop" || s == "down" || s == "on" || s == "off" || s == "toggle" || s == "dim"
}
func validTime(s string, optional bool) bool {
	if s == "" {
		return optional
	}
	_, e := time.Parse("15:04", s)
	return e == nil && len(s) == 5
}
func validate(c Config) error {
	if c.PreferredController != "" && !regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`).MatchString(c.PreferredController) {
		return errors.New("invalid preferred TellStick serial")
	}
	if err := validateControlOrder(c); err != nil {
		return err
	}
	if !validIcon(c.SchedulerIcon) {
		return errors.New("invalid scheduler symbol")
	}
	if c.Port < 1024 || c.Port > 65535 || c.SessionDays < 1 || c.SessionDays > 90 {
		return errors.New("port must be 1024..65535; session days 1..90")
	}
	if _, e := time.LoadLocation(c.Timezone); e != nil && c.Timezone != "router" {
		return errors.New("unknown time zone")
	}
	if c.Latitude < -90 || c.Latitude > 90 || c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("invalid coordinates")
	}
	if len(c.Networks) == 0 || len(c.Networks) > 16 {
		return errors.New("select LAN and/or VPN networks")
	}
	for _, n := range c.Networks {
		if !idRE.MatchString(n) || n == "wan" || n == "wan6" {
			return errors.New("invalid or WAN network")
		}
	}
	if len(c.Devices) > 64 || len(c.Groups) > 32 || len(c.Schedules) > 128 {
		return errors.New("configuration limit exceeded")
	}
	ids := map[string]bool{}
	radios := map[int]bool{}
	for _, d := range c.Devices {
		if !validIcon(d.Icon) {
			return errors.New("invalid device symbol")
		}
		if !idRE.MatchString(d.ID) || ids[d.ID] || len(d.Name) < 1 || len(d.Name) > 80 || d.RadioID < 1 || d.RadioID > 65535 || radios[d.RadioID] {
			return errors.New("invalid or duplicate device")
		}
		if err := validateDeviceType(d); err != nil {
			return err
		}
		ids[d.ID] = true
		radios[d.RadioID] = true
	}
	for _, g := range c.Groups {
		if !validIcon(g.Icon) {
			return errors.New("invalid group symbol")
		}
		if !idRE.MatchString(g.ID) || ids[g.ID] || len(g.Name) < 1 || len(g.Name) > 80 || len(g.Devices) == 0 || len(g.Devices) > 64 {
			return errors.New("invalid group")
		}
		if c.targetType(g.ID) == "" {
			return errors.New("group members must have the same device type")
		}
		members := map[string]bool{}
		for _, id := range g.Devices {
			if members[id] {
				return errors.New("group contains a duplicate device")
			}
			members[id] = true
			found := false
			for _, d := range c.Devices {
				if d.ID == id {
					found = true
				}
			}
			if !found {
				return errors.New("group references missing device")
			}
		}
		ids[g.ID] = true
	}
	seen := map[string]bool{}
	for _, s := range c.Schedules {
		if !idRE.MatchString(s.ID) || seen[s.ID] || !ids[s.Target] || !c.supportsAction(s.Target, s.Action) || len(s.Days) == 0 || len(s.Days) > 7 || s.Offset < -720 || s.Offset > 720 {
			return errors.New("invalid schedule")
		}
		seen[s.ID] = true
		for _, d := range s.Days {
			if d < 0 || d > 6 {
				return errors.New("invalid weekday")
			}
		}
		if s.Event != "time" && s.Event != "sunrise" && s.Event != "sunset" && s.Event != "dawn" && s.Event != "dusk" {
			return errors.New("invalid solar event")
		}
		if !validTime(s.Time, s.Event != "time") || !validTime(s.Earliest, true) || !validTime(s.Latest, true) || !validTime(s.Fallback, true) {
			return errors.New("use HH:MM times")
		}
		if s.Earliest != "" && s.Latest != "" && s.Earliest > s.Latest {
			return errors.New("earliest must not follow latest")
		}
	}
	return nil
}
func loadConfig() (Config, error) {
	f, e := os.Open(base + "/config.json")
	if e != nil {
		return Config{}, e
	}
	defer f.Close()
	return parseConfig(f)
}
func parseConfig(r io.Reader) (Config, error) {
	var c Config
	e := decode(r, &c)
	c.Controller = "" // Serial selection is automatic, including after hardware replacement.
	if e == nil {
		e = validate(c)
	}
	return c, e
}
func decode(r io.Reader, v any) error {
	d := json.NewDecoder(io.LimitReader(r, 65537))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func answer(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	answer(w, status, map[string]string{"error": msg})
}
func (s *Server) session(r *http.Request) *Session {
	c, e := r.Cookie(cookieName)
	if e != nil || len(c.Value) != 64 {
		return nil
	}
	h := hash(c.Value)
	for _, v := range s.sessions {
		if v.Hash == h && v.Expires > time.Now().Unix() && v.Revision == s.cred.Revision {
			return &v
		}
	}
	return nil
}
func (s *Server) persistSessions() {
	if e := atomicJSON(base+"/state/sessions.json", s.sessions); e != nil {
		log.Print("session persistence failed")
	}
}
func (s *Server) permittedAttempt(ip string, now time.Time) bool {
	if len(s.attempts) > 128 {
		s.attempts = map[string][]time.Time{}
	}
	for _, key := range []string{"*", ip} {
		a := s.attempts[key]
		fresh := a[:0]
		for _, t := range a {
			if now.Sub(t) < time.Minute {
				fresh = append(fresh, t)
			}
		}
		s.attempts[key] = fresh
		limit := 5
		if key == "*" {
			limit = 20
		}
		if len(fresh) >= limit {
			return false
		}
	}
	s.attempts["*"] = append(s.attempts["*"], now)
	s.attempts[ip] = append(s.attempts[ip], now)
	return true
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" && r.Method != "POST" {
		fail(w, 405, "method not allowed")
		return
	}
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "application/json" {
			fail(w, 403, "request protection required")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "https://"+r.Host {
			fail(w, 403, "origin rejected")
			return
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.URL.Path == "/api/login" && r.Method == "POST" {
		if !s.permittedAttempt(r.Header.Get("X-Client-IP"), time.Now()) {
			fail(w, 429, "too many attempts; wait one minute")
			return
		}
		var v struct {
			User     string `json:"user"`
			Password string `json:"password"`
			Remember bool   `json:"remember"`
		}
		if decode(r.Body, &v) != nil || utf8.RuneCountInString(v.Password) > 256 {
			fail(w, 400, "invalid login")
			return
		}
		got := passwordHash(v.Password, s.cred.Salt)
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.cred.Hash)) != 1 || v.User != s.cred.User || s.cred.Hash == "" {
			fail(w, 401, "incorrect username or password")
			return
		}
		now := time.Now()
		duration := 12 * time.Hour
		if v.Remember {
			duration = time.Duration(s.config.SessionDays) * 24 * time.Hour
		}
		t := token()
		session := Session{Hash: hash(t), CSRF: token(), Expires: now.Add(duration).Unix(), Revision: s.cred.Revision, Created: now.Unix(), Device: r.UserAgent()}
		if len(session.Device) > 160 {
			session.Device = session.Device[:160]
		}
		fresh := []Session{}
		for _, x := range s.sessions {
			if x.Expires > now.Unix() && x.Revision == s.cred.Revision {
				fresh = append(fresh, x)
			}
		}
		if len(fresh) >= 32 {
			fresh = fresh[len(fresh)-31:]
		}
		s.sessions = append(fresh, session)
		s.persistSessions()
		cookie := &http.Cookie{Name: cookieName, Value: t, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
		if v.Remember {
			cookie.MaxAge = int(duration.Seconds())
			cookie.Expires = now.Add(duration)
		}
		http.SetCookie(w, cookie)
		answer(w, 200, map[string]string{"csrf": session.CSRF})
		return
	}
	session := s.session(r)
	if session == nil {
		fail(w, 401, "login required")
		return
	}
	if r.Method == "POST" && subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), []byte(session.CSRF)) != 1 {
		fail(w, 403, "invalid request token")
		return
	}
	switch r.URL.Path {
	case "/api/state":
		if r.Method != "GET" {
			fail(w, 405, "GET required")
			return
		}
		devices := []map[string]any{}
		for _, d := range s.config.Devices {
			devices = append(devices, map[string]any{"id": d.ID, "name": d.Name, "icon": d.Icon, "type": d.deviceType(), "state": s.states[d.ID]})
		}
		groups := []map[string]any{}
		for _, g := range s.config.Groups {
			groups = append(groups, map[string]any{"id": g.ID, "name": g.Name, "icon": g.Icon, "devices": g.Devices, "type": s.config.targetType(g.ID)})
		}
		answer(w, 200, map[string]any{"devices": devices, "groups": groups, "group_states": s.groupStates(), "csrf": session.CSRF, "queue": len(s.queue), "position_feedback": false, "scheduler": s.schedulerStatus(time.Now()), "scheduler_icon": s.config.SchedulerIcon, "control_order": orderedControlKeys(s.config)})
	case "/api/scheduler":
		if r.Method != "POST" {
			fail(w, 405, "POST required")
			return
		}
		var v struct {
			Action string `json:"action"`
		}
		if decode(r.Body, &v) != nil || (v.Action != "start" && v.Action != "stop") {
			fail(w, 400, "choose start or stop")
			return
		}
		if e := s.setScheduler(v.Action == "stop", time.Now()); e != nil {
			fail(w, 500, "could not save scheduler state")
			return
		}
		answer(w, 200, s.schedulerStatus(time.Now()))
	case "/api/command":
		if r.Method != "POST" {
			fail(w, 405, "POST required")
			return
		}
		var v struct {
			Target string `json:"target"`
			Action string `json:"action"`
		}
		if decode(r.Body, &v) != nil || !validAction(v.Action) {
			fail(w, 400, "invalid command")
			return
		}
		if e := s.enqueue(v.Target, v.Action, "manual"); e != nil {
			fail(w, 400, e.Error())
			return
		}
		answer(w, 202, map[string]bool{"accepted": true})
	case "/api/logout":
		if r.Method != "POST" {
			fail(w, 405, "POST required")
			return
		}
		for i, v := range s.sessions {
			if v.Hash == session.Hash {
				s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
				break
			}
		}
		s.persistSessions()
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		answer(w, 200, map[string]bool{"ok": true})
	default:
		fail(w, 404, "not found")
	}
}
func (s *Server) enqueue(target, action, source string) error {
	if !s.config.supportsAction(target, action) {
		return errors.New("action is not supported by this target")
	}
	dimmer := s.config.targetType(target) == "dimmer"
	ids := []string{}
	for _, d := range s.config.Devices {
		if d.ID == target {
			ids = append(ids, d.ID)
		}
	}
	for _, g := range s.config.Groups {
		if g.ID == target {
			ids = append(ids, g.Devices...)
		}
	}
	if len(ids) == 0 {
		return errors.New("unknown target")
	}
	// Toggle operations cannot replace one another: every accepted press matters.
	// Bound pending presses, checking the whole group before changing any queue.
	if dimmer {
		for _, id := range ids {
			pending := 0
			for _, queued := range s.queue {
				if queued.Device == id {
					pending++
				}
			}
			if pending >= 4 {
				return errors.New("wait for pending dimmer commands")
			}
		}
	}
	request := s.beginRequest(target, action, ids)
	for _, id := range ids {
		q := []Command{}
		for _, c := range s.queue {
			if c.Device != id || dimmer {
				q = append(q, c)
			} else {
				s.finishCommand(c, "Superseded", time.Now())
			}
		}
		cmd := Command{Device: id, Action: action, Source: source, Expires: time.Now().Add(30 * time.Second), Target: target, Request: request}
		if action == "stop" {
			q = append([]Command{cmd}, q...)
		} else {
			q = append(q, cmd)
		}
		s.queue = q
		s.latestDevice[id] = request
		s.states[id] = DeviceState{action, "Queued", time.Now().Unix()}
	}
	select {
	case s.wake <- true:
	default:
	}
	return nil
}
func (s *Server) send(c Command) error {
	if !s.config.supportsAction(c.Device, c.Action) {
		return errors.New("action is not supported by this device")
	}
	unlock, err := radioLock()
	if err != nil {
		return errors.New("radio is being reconfigured")
	}
	defer unlock()
	if !c.Expires.IsZero() && time.Now().After(c.Expires) {
		return errors.New("command expired")
	}
	var active map[string]string
	if readJSON(run+"/controller.json", &active) != nil || active["serial"] == "" {
		return errors.New("no TellStick connected")
	}
	var id int
	radioAction := c.Action
	for _, d := range s.config.Devices {
		if d.ID == c.Device {
			id = d.RadioID
			radioAction = d.radioAction(c.Action)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, e := exec.CommandContext(ctx, "/usr/libexec/home-controller-radio", strconv.Itoa(id), radioAction).CombinedOutput()
	if e != nil || strings.TrimSpace(string(out)) != "0" {
		return errors.New("radio command failed")
	}
	return nil
}
func (s *Server) worker() {
	for range s.wake {
		for {
			s.mu.Lock()
			if len(s.queue) == 0 {
				s.mu.Unlock()
				break
			}
			c := s.queue[0]
			s.queue = s.queue[1:]
			s.mu.Unlock()
			var e error
			if time.Now().After(c.Expires) {
				e = errors.New("command expired")
			} else {
				e = s.sender(c)
			}
			result := "Sent (position unknown)"
			if e != nil {
				result = e.Error()
			}
			s.mu.Lock()
			s.finishCommand(c, result, time.Now())
			s.history = append(s.history, Event{time.Now().Unix(), c.Device, c.Action, c.Source, result})
			if len(s.history) > 100 {
				s.history = s.history[len(s.history)-100:]
			}
			s.writeStatus(time.Now())
			s.mu.Unlock()
		}
	}
}
func clockOn(day time.Time, hhmm string) time.Time {
	p := strings.Split(hhmm, ":")
	h, _ := strconv.Atoi(p[0])
	m, _ := strconv.Atoi(p[1])
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location())
}
func solar(c Config, s Schedule, day time.Time) (time.Time, error) {
	rise := "rise"
	if s.Event == "sunset" || s.Event == "dusk" {
		rise = "set"
	}
	kind := "daylight"
	if s.Event == "dawn" || s.Event == "dusk" {
		kind = "civil"
	}
	coord := func(v float64, pos, neg string) string {
		if v < 0 {
			return fmt.Sprintf("%.6f%s", -v, neg)
		}
		return fmt.Sprintf("%.6f%s", v, pos)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if day.Year() < 2000 || day.Year() > 2099 {
		return time.Time{}, errors.New("sunwait supports years 2000..2099")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/sunwait", "list", "1", "gmt", kind, rise, "y", strconv.Itoa(day.Year()%100), "m", strconv.Itoa(int(day.Month())), "d", strconv.Itoa(day.Day()), coord(c.Latitude, "N", "S"), coord(c.Longitude, "E", "W"))
	cmd.Env = []string{"TZ=UTC", "PATH=/usr/bin:/bin"}
	out, e := cmd.Output()
	if e != nil {
		return time.Time{}, errors.New("sunwait unavailable")
	}
	v := strings.TrimSpace(string(out))
	if !validTime(v, false) {
		if strings.HasPrefix(v, "--:--") {
			return time.Time{}, errNoSolarEvent
		}
		return time.Time{}, errors.New("invalid sunwait response")
	}
	utcDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	t := clockOn(utcDay, v).In(day.Location())
	if t.Day() != day.Day() {
		if t.Before(day) {
			t = t.Add(24 * time.Hour)
		} else {
			t = t.Add(-24 * time.Hour)
		}
	}
	return t, nil
}
func (s *Server) eventTime(rule Schedule, day time.Time) (time.Time, error) {
	var t time.Time
	var e error
	if rule.Event == "time" {
		t = clockOn(day, rule.Time)
	} else {
		t, e = s.solar(rule, day)
		if e != nil {
			if rule.Fallback == "" || !errors.Is(e, errNoSolarEvent) {
				return t, e
			}
			t = clockOn(day, rule.Fallback)
		} else {
			t = t.Add(time.Duration(rule.Offset) * time.Minute)
		}
	}
	if rule.Earliest != "" && t.Before(clockOn(day, rule.Earliest)) {
		t = clockOn(day, rule.Earliest)
	}
	if rule.Latest != "" && t.After(clockOn(day, rule.Latest)) {
		t = clockOn(day, rule.Latest)
	}
	return t, nil
}
func (s *Server) due(last, now time.Time, seen map[string]bool) {
	if s.schedulerStopped {
		return
	}
	if s.resumeAt.After(last) {
		last = s.resumeAt
	}
	if now.Before(last) || now.Sub(last) > 90*time.Second {
		return
	}
	loc, _ := time.LoadLocation(s.config.Timezone)
	today := now.In(loc)
	for _, rule := range s.config.Schedules {
		if !rule.Enabled {
			continue
		}
		for delta := -1; delta <= 1; delta++ {
			day := time.Date(today.Year(), today.Month(), today.Day()+delta, 0, 0, 0, 0, loc)
			match := false
			for _, d := range rule.Days {
				if d == int(day.Weekday()) {
					match = true
				}
			}
			if !match {
				continue
			}
			key := rule.ID + day.Format("2006-01-02")
			if seen[key] {
				continue
			}
			t, e := s.eventTime(rule, day)
			if e == nil && t.After(last) && !t.After(now) {
				seen[key] = true
				s.enqueue(rule.Target, rule.Action, "schedule")
			}
		}
	}
}
func (s *Server) scheduler() {
	last := time.Now()
	seen := map[string]bool{}
	for now := range time.Tick(5 * time.Second) {
		if now.Year() < 2025 {
			last = now
			continue
		}
		s.mu.Lock()
		s.due(last, now, seen)
		s.writeStatus(now)
		s.mu.Unlock()
		last = now
		if len(seen) > 1024 {
			seen = map[string]bool{}
		}
	}
}
func serve() error {
	c, e := loadConfig()
	if e != nil {
		return e
	}
	if c.Timezone == "router" {
		if e := readJSON(run+"/timezone.json", &c.Timezone); e != nil {
			return errors.New("router time zone unavailable")
		}
		if _, e = time.LoadLocation(c.Timezone); e != nil {
			return e
		}
	}
	s := &Server{config: c, states: map[string]DeviceState{}, wake: make(chan bool, 1), attempts: map[string][]time.Time{}}
	s.schedulerFile = base + "/state/scheduler.json"
	if e := s.loadScheduler(); e != nil {
		s.schedulerStopped = true
		log.Print("scheduler state unreadable; scheduler stopped")
	}
	readJSON(base+"/credential.json", &s.cred)
	readJSON(base+"/state/sessions.json", &s.sessions)
	s.sender = s.send
	type solarResult struct {
		at  time.Time
		err error
	}
	cache := map[string]solarResult{}
	s.solar = func(r Schedule, d time.Time) (time.Time, error) {
		key := r.Event + d.Format("2006-01-02")
		if v, ok := cache[key]; ok {
			return v.at, v.err
		}
		if len(cache) > 128 {
			cache = map[string]solarResult{}
		}
		at, e := solar(c, r, d)
		cache[key] = solarResult{at, e}
		return at, e
	}
	os.MkdirAll(run, 0700)
	os.Remove(run + "/api.sock")
	ln, e := net.Listen("unix", run+"/api.sock")
	if e != nil {
		return e
	}
	os.Chmod(run+"/api.sock", 0600)
	if e := atomicJSON(run+"/status.json", map[string]any{"devices": s.states, "history": []Event{}, "queue": 0, "scheduler": s.schedulerStatus(time.Now())}); e != nil {
		ln.Close()
		return e
	}
	go s.worker()
	go s.scheduler()
	return (&http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, MaxHeaderBytes: 8192}).Serve(ln)
}
func proxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'none'; form-action 'self'")
	if r.URL.Path == "/api/page" && r.Method == "GET" {
		b, e := os.ReadFile("/usr/share/home-controller/page.html")
		if e != nil {
			fail(w, 503, "page unavailable")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
		return
	}
	if !map[string]bool{"/api/login": true, "/api/logout": true, "/api/state": true, "/api/command": true, "/api/scheduler": true}[r.URL.Path] {
		fail(w, 404, "not found")
		return
	}
	body, e := io.ReadAll(io.LimitReader(r.Body, 4097))
	if e != nil || len(body) > 4096 {
		fail(w, 413, "request too large")
		return
	}
	req, e := http.NewRequest(r.Method, "http://unix"+r.URL.Path, bytes.NewReader(body))
	if e != nil {
		fail(w, 400, "invalid request")
		return
	}
	req.Host = r.Host
	for _, h := range []string{"Cookie", "Content-Type", "Origin", "Authorization", "User-Agent"} {
		req.Header.Set(h, r.Header.Get(h))
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	req.Header.Set("X-Client-IP", ip)
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", run+"/api.sock")
	}}}
	resp, e := client.Do(req)
	if e != nil {
		fail(w, 503, "controller unavailable")
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
func main() {
	if os.Getenv("GATEWAY_INTERFACE") != "" {
		cgi.Serve(http.HandlerFunc(proxy))
		return
	}
	if len(os.Args) < 2 {
		log.Fatal("expected serve, rpc, or configure")
	}
	var e error
	switch os.Args[1] {
	case "serve":
		e = serve()
	case "rpc":
		e = rpc()
	case "configure":
		e = configure()
	case "usb-change":
		e = usbChange()
	default:
		e = errors.New("unknown operation")
	}
	if e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
