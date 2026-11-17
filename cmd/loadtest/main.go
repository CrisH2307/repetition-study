// Command loadtest measures the non-AI read paths (D53, gate G3: p95 under 50 ms
// at 500 concurrent users). Stdlib only. It refuses any host except localhost
// unless -allow-host names that exact host. Never aim it at production.
//
//	loadtest -base http://localhost:8080 -tokens invites.txt -users 500 -duration 30s
//
// invites.txt holds one invite link or bare token per line. Simulated user i
// signs in with line i (wrapping), so use one line per user for a fair test.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	base := flag.String("base", "http://localhost:8080", "API address")
	tokens := flag.String("tokens", "", "file with one invite link or token per line (required)")
	users := flag.Int("users", 500, "concurrent simulated users")
	dur := flag.Duration("duration", 30*time.Second, "how long to keep reading")
	allow := flag.String("allow-host", "", "non-localhost host name you are allowed to test (staging copy only)")
	flag.Parse()
	if err := checkHost(*base, *allow); err != nil {
		log.Fatal(err)
	}
	toks, err := readTokens(*tokens)
	if err != nil {
		log.Fatal(err)
	}
	if *users < 1 {
		log.Fatal("-users must be 1 or more")
	}
	res, err := run(context.Background(), *base, toks, *users, *dur)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(res.report(*users, *dur))
	if res.p(0.95) >= 50*time.Millisecond || res.errs > 0 {
		os.Exit(1) // bar missed: p95 under 50 ms and no errors
	}
}

// checkHost allows localhost, 127.0.0.1, [::1], or exactly the -allow-host name.
func checkHost(base, allow string) error {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("bad -base %q", base)
	}
	h := strings.ToLower(u.Hostname())
	if h == "localhost" || h == "127.0.0.1" || h == "::1" || (allow != "" && h == strings.ToLower(allow)) {
		return nil
	}
	return fmt.Errorf("refusing to load %q: only localhost, or a host named with -allow-host", h)
}

func readTokens(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("-tokens is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if t := token(sc.Text()); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s has no tokens", path)
	}
	return out, sc.Err()
}

// token takes a bare token or a link like https://host/invite#TOKEN.
func token(line string) string {
	line = strings.TrimSpace(line)
	if i := strings.LastIndex(line, "#"); i >= 0 {
		return line[i+1:]
	}
	return line
}

type result struct {
	lat  []time.Duration
	errs int
	per  map[string][]time.Duration
}

func (r *result) p(q float64) time.Duration { return pct(r.lat, q) }

func pct(d []time.Duration, q float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := int(float64(len(s))*q+0.9999999) - 1 // nearest rank
	if i < 0 {
		i = 0
	}
	return s[i]
}

func (r *result) report(users int, dur time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "users %d, duration %s, requests %d, errors %d, rate %.0f req/s\n", users, dur, len(r.lat), r.errs, float64(len(r.lat))/dur.Seconds())
	line := func(name string, d []time.Duration) {
		fmt.Fprintf(&b, "%-12s n=%-7d p50=%-9s p95=%-9s p99=%s\n", name, len(d), pct(d, .5), pct(d, .95), pct(d, .99))
	}
	line("ALL", r.lat)
	names := make([]string, 0, len(r.per))
	for k := range r.per {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		line(k, r.per[k])
	}
	if r.p(.95) < 50*time.Millisecond && r.errs == 0 {
		b.WriteString("PASS: p95 under 50 ms, no errors\n")
	} else {
		b.WriteString("FAIL: bar is p95 under 50 ms and no errors\n")
	}
	return b.String()
}

type user struct {
	c      *http.Client
	course string
	paths  []string
}

func newClient(base string) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 2}}
}

func call(c *http.Client, method, u string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u, rd)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if out != nil && resp.StatusCode < 300 {
		json.Unmarshal(b, out)
	}
	return resp.StatusCode, nil
}

// setup signs in and makes sure the user has one course with a few concepts.
// It is not timed.
func setup(base, tok string) (*user, error) {
	c := newClient(base)
	if st, err := call(c, "POST", base+"/v1/auth/redeem", map[string]string{"token": tok}, nil); err != nil || st != 200 {
		return nil, fmt.Errorf("sign-in failed (status %d, err %v)", st, err)
	}
	var courses []struct {
		ID string `json:"id"`
	}
	if _, err := call(c, "GET", base+"/v1/courses", nil, &courses); err != nil {
		return nil, err
	}
	if len(courses) == 0 {
		var made struct {
			ID string `json:"id"`
		}
		in := map[string]any{"name": "Load test course", "weekly_hours": 3, "level": "beginner"}
		if st, err := call(c, "POST", base+"/v1/courses", in, &made); err != nil || st != 201 || made.ID == "" {
			return nil, fmt.Errorf("could not make a course (status %d, err %v)", st, err)
		}
		courses = append(courses, made)
		for _, n := range []string{"Limits", "Derivatives", "Integrals"} {
			call(c, "POST", base+"/v1/courses/"+made.ID+"/concepts", map[string]string{"name": n}, nil)
		}
	}
	id := courses[0].ID
	return &user{c: c, course: id}, nil
}

func run(ctx context.Context, base string, toks []string, n int, dur time.Duration) (*result, error) {
	base = strings.TrimRight(base, "/")
	us := make([]*user, n)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	sem := make(chan struct{}, 50) // sign in gently so setup is not the test
	for i := range us {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			u, err := setup(base, toks[i%len(toks)])
			<-sem
			mu.Lock()
			defer mu.Unlock()
			if err != nil && first == nil {
				first = err
			}
			us[i] = u
		}()
	}
	wg.Wait()
	if first != nil {
		return nil, first
	}
	res := &result{per: map[string][]time.Duration{}}
	end := time.Now().Add(dur)
	for _, u := range us {
		wg.Add(1)
		go func() {
			defer wg.Done()
			paths := map[string]string{
				"today":       "/v1/today",
				"week":        "/v1/week?week=" + monday(),
				"courses":     "/v1/courses",
				"course_view": "/v1/courses/" + u.course + "/view",
			}
			order := []string{"today", "week", "courses", "course_view"}
			for k := 0; time.Now().Before(end) && ctx.Err() == nil; k++ {
				name := order[k%len(order)]
				t := time.Now()
				st, err := call(u.c, "GET", base+paths[name], nil, nil)
				d := time.Since(t)
				mu.Lock()
				res.lat = append(res.lat, d)
				res.per[name] = append(res.per[name], d)
				if err != nil || st != 200 {
					res.errs++
				}
				mu.Unlock()
				time.Sleep(50 * time.Millisecond) // a person pauses; no hammering
			}
		}()
	}
	wg.Wait()
	return res, nil
}

// monday is the Monday of this week, the date /v1/week expects.
func monday() string {
	t := time.Now().UTC()
	return t.AddDate(0, 0, -(int(t.Weekday())+6)%7).Format("2006-01-02")
}
