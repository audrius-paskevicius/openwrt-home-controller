package main

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func exampleConfig() Config {
	return Config{Enabled: true, Port: 8443, Networks: []string{"lan"}, Timezone: "Europe/Stockholm", SessionDays: 30, Devices: []Device{{ID: "blind", Name: "Example blind", RadioID: 1, House: 1234, Unit: 0}}, Groups: []Group{{"all", "All", []string{"blind"}, ""}}}
}
func TestAutomaticControllerSelection(t *testing.T) {
	a := map[string]string{"serial": "TEST_A"}
	b := map[string]string{"serial": "TEST_B"}
	for _, tc := range []struct {
		devices        []map[string]string
		previous, want string
	}{
		{nil, "TEST_A", ""},
		{[]map[string]string{b}, "TEST_A", "TEST_B"},
		{[]map[string]string{a, b}, "", "TEST_A"},
		{[]map[string]string{b, a}, "TEST_A", "TEST_A"},
	} {
		if got := selectController(tc.devices, tc.previous)["serial"]; got != tc.want {
			t.Fatalf("selection: got %q, want %q", got, tc.want)
		}
	}
}
func TestLegacyControllerConfiguration(t *testing.T) {
	c := exampleConfig()
	c.Controller = "DISCONNECTED_OLD_DEVICE"
	b, _ := json.Marshal(c)
	got, err := parseConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if got.Controller != "" || len(got.Devices) != len(c.Devices) {
		t.Fatal("legacy pin retained or devices lost")
	}
	b, _ = json.Marshal(got)
	if bytes.Contains(b, []byte(`"controller"`)) {
		t.Fatal("obsolete setting still published")
	}
}
func testServer() *Server {
	c := exampleConfig()
	s := &Server{config: c, cred: Credential{Revision: "revision"}, states: map[string]DeviceState{}, wake: make(chan bool, 1), attempts: map[string][]time.Time{}}
	s.solar = func(_ Schedule, d time.Time) (time.Time, error) { return clockOn(d, "06:00"), nil }
	return s
}
func TestPasswordKDF(t *testing.T) {
	want, e := pbkdf2.Key(sha256.New, "test-password", []byte("salt"), 210000, 32)
	if e != nil {
		t.Fatal(e)
	}
	if got := passwordHash("test-password", "salt"); got != hex.EncodeToString(want) {
		t.Fatal("KDF mismatch")
	}
}
func TestSharedPasswordChoice(t *testing.T) {
	for _, p := range []string{"x", "short", " å ", strings.Repeat("å", 256)} {
		if err := validateSharedAccount("manual", p); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"", strings.Repeat("x", 257), strings.Repeat("å", 257)} {
		if validateSharedAccount("manual", p) == nil {
			t.Fatal("accepted empty or excessive password")
		}
	}
	s := testServer()
	s.cred = Credential{User: "manual", Salt: "test-salt", Hash: passwordHash("short", "test-salt"), Revision: "test"}
	// Exercise the real login handler without touching persistent session storage.
	w := request(s, "POST", "/api/login", `{"user":"manual","password":"wrong"}`, map[string]string{"Content-Type": "application/json"})
	if w.Code != 401 {
		t.Fatalf("incorrect password: %d", w.Code)
	}
}
func TestConfigurationValidation(t *testing.T) {
	c := exampleConfig()
	if e := validate(c); e != nil {
		t.Fatal(e)
	}
	cases := []func(*Config){func(c *Config) { c.Networks = []string{"wan"} }, func(c *Config) { c.Devices[0].Unit = -1 }, func(c *Config) { c.Devices[0].House = 0 }, func(c *Config) { c.Groups[0].Devices = []string{"missing"} }, func(c *Config) { c.SessionDays = 999 }, func(c *Config) { c.Timezone = "nonsense" }, func(c *Config) { c.Latitude = 91 }}
	for i, change := range cases {
		c := exampleConfig()
		change(&c)
		if validate(c) == nil {
			t.Errorf("accepted invalid configuration %d", i)
		}
	}
}
func request(s *Server, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "router:8443"
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestControlsAuthorization(t *testing.T) {
	s := testServer()
	tok := strings.Repeat("a", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	headers := map[string]string{"Cookie": cookieName + "=" + tok, "Content-Type": "application/json", "X-Home-Controller": "1", "Authorization": "Bearer csrf"}
	if request(s, "GET", "/api/state", "", nil).Code != 401 {
		t.Fatal("unauthenticated state")
	}
	for _, path := range []string{"/admin/save", "/api/save", "/api/password"} {
		if request(s, "POST", path, `{}`, headers).Code != 404 {
			t.Fatal("control management route")
		}
	}
	bad := map[string]string{}
	for k, v := range headers {
		bad[k] = v
	}
	bad["Origin"] = "https://evil.invalid"
	if request(s, "POST", "/api/command", `{"target":"blind","action":"up"}`, bad).Code != 403 {
		t.Fatal("cross origin accepted")
	}
	bad["Origin"] = "https://router:8443"
	bad["Authorization"] = "Bearer wrong"
	if request(s, "POST", "/api/command", `{}`, bad).Code != 403 {
		t.Fatal("CSRF accepted")
	}
	if request(s, "GET", "/api/command", "", headers).Code != 405 {
		t.Fatal("GET command")
	}
	if request(s, "POST", "/api/command", `{"target":"blind","action":"up","shell":"bad"}`, headers).Code != 400 {
		t.Fatal("unexpected fields accepted")
	}
	if request(s, "POST", "/api/command", `{"target":"blind","action":"up"}`, headers).Code != 202 {
		t.Fatal("valid command rejected")
	}
	s.cred.Revision = "new"
	if request(s, "GET", "/api/state", "", headers).Code != 401 {
		t.Fatal("revoked session accepted")
	}
}
func TestPriorityAndDeduplication(t *testing.T) {
	s := testServer()
	s.enqueue("blind", "down", "test")
	s.enqueue("all", "up", "test")
	s.enqueue("blind", "stop", "test")
	if len(s.queue) != 1 || s.queue[0].Action != "stop" {
		t.Fatal(s.queue)
	}
	if s.enqueue("missing", "up", "test") == nil {
		t.Fatal("missing target")
	}
}
func TestScheduleDowntimeAndManual(t *testing.T) {
	s := testServer()
	loc, _ := time.LoadLocation(s.config.Timezone)
	now := time.Date(2026, 11, 9, 8, 0, 2, 0, loc)
	s.config.Schedules = []Schedule{{ID: "morning", Enabled: true, Target: "blind", Action: "up", Days: []int{1}, Event: "time", Time: "08:00"}}
	seen := map[string]bool{}
	s.due(now.Add(-time.Hour), now, seen)
	if len(s.queue) != 0 {
		t.Fatal("replayed downtime")
	}
	s.enqueue("blind", "down", "manual")
	s.due(now.Add(-5*time.Second), now, seen)
	if len(s.queue) != 1 || s.queue[0].Action != "up" {
		t.Fatal("manual command disabled future schedule")
	}
	s.queue = nil
	s.due(now.Add(-5*time.Second), now, seen)
	if len(s.queue) != 0 {
		t.Fatal("duplicate event")
	}
}
func TestSolarBounds(t *testing.T) {
	s := testServer()
	loc, _ := time.LoadLocation(s.config.Timezone)
	day := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	r := Schedule{Event: "sunrise", Offset: -20, Earliest: "07:30", Latest: "09:00"}
	at, e := s.eventTime(r, day)
	if e != nil || at.Format("15:04") != "07:30" {
		t.Fatal(at, e)
	}
}

func TestSolarFallback(t *testing.T) {
	s := testServer()
	day := time.Now()
	r := Schedule{Event: "sunrise", Fallback: "08:00"}
	s.solar = func(Schedule, time.Time) (time.Time, error) { return time.Time{}, errNoSolarEvent }
	at, e := s.eventTime(r, day)
	if e != nil || at.Format("15:04") != "08:00" {
		t.Fatal(at, e)
	}
	s.solar = func(Schedule, time.Time) (time.Time, error) { return time.Time{}, errors.New("sunwait unavailable") }
	if _, e = s.eventTime(r, day); e == nil {
		t.Fatal("technical error must not become a solar fallback")
	}
}
func TestDST(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Stockholm")
	for _, date := range []string{"2026-03-29", "2026-10-25"} {
		d, _ := time.ParseInLocation("2006-01-02", date, loc)
		if clockOn(d, "08:00").Hour() != 8 {
			t.Fatal("wall clock shifted")
		}
	}
}
func TestRateLimit(t *testing.T) {
	s := testServer()
	now := time.Now()
	for i := 0; i < 5; i++ {
		if !s.permittedAttempt("ip", now) {
			t.Fatal(i)
		}
	}
	if s.permittedAttempt("ip", now) {
		t.Fatal("no limit")
	}
	if !s.permittedAttempt("ip", now.Add(61*time.Second)) {
		t.Fatal("limit did not expire")
	}
}
func TestStrictJSON(t *testing.T) {
	for _, v := range []string{`{} {}`, `{"unexpected":1}`, strings.Repeat("x", 70000)} {
		var c Config
		if decode(strings.NewReader(v), &c) == nil {
			t.Fatal("accepted", v[:2])
		}
	}
}
func TestNoPrivateDataInState(t *testing.T) {
	s := testServer()
	tok := strings.Repeat("a", 64)
	s.sessions = []Session{{Hash: hash(tok), CSRF: "csrf", Expires: time.Now().Add(time.Hour).Unix(), Revision: "revision"}}
	w := request(s, "GET", "/api/state", "", map[string]string{"Cookie": cookieName + "=" + tok})
	var data map[string]any
	if json.Unmarshal(w.Body.Bytes(), &data) != nil || w.Code != 200 {
		t.Fatal(w.Body)
	}
	for _, secret := range []string{"house", "radio_id", "latitude", "salt", "revision"} {
		if bytes.Contains(w.Body.Bytes(), []byte(`"`+secret+`"`)) {
			t.Fatal("leaked", secret)
		}
	}
}
func TestCookiePolicy(t *testing.T) {
	c := http.Cookie{Name: cookieName, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Path: "/"}
	if !strings.Contains(c.String(), "SameSite=Strict") {
		t.Fatal(c)
	}
}

func TestInstalledSunwait(t *testing.T) {
	if !exists("/usr/bin/sunwait") {
		t.Skip("target integration test requires sunwait")
	}
	c := exampleConfig()
	c.Latitude = 51.4779
	c.Longitude = 0
	loc, _ := time.LoadLocation("Europe/Stockholm")
	r := Schedule{Event: "sunrise"}
	for _, date := range []struct {
		month time.Month
		day   int
		delta time.Duration
	}{{3, 28, time.Hour}, {10, 24, -time.Hour}} {
		d := time.Date(2026, date.month, date.day, 0, 0, 0, 0, loc)
		a, e := solar(c, r, d)
		if e != nil {
			t.Fatal(e)
		}
		b, e := solar(c, r, d.AddDate(0, 0, 1))
		if e != nil {
			t.Fatal(e)
		}
		utcDiff := b.Sub(a) - 24*time.Hour
		if utcDiff < -10*time.Minute || utcDiff > 10*time.Minute {
			t.Fatal("UTC discontinuity", a, b)
		}
		wallDiff := time.Duration((b.Hour()-a.Hour())*60+b.Minute()-a.Minute()) * time.Minute
		if wallDiff-date.delta < -10*time.Minute || wallDiff-date.delta > 10*time.Minute {
			t.Fatal("DST conversion", a, b)
		}
	}
	c.Latitude = 80
	for _, month := range []time.Month{6, 12} {
		if _, e := solar(c, r, time.Date(2026, month, 21, 0, 0, 0, 0, loc)); e == nil {
			t.Fatal("invented polar solar event")
		}
	}
}
