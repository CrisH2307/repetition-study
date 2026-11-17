package main

import (
	"testing"
	"time"
)

func TestCheckHost(t *testing.T) {
	ok := []struct{ base, allow string }{
		{"http://localhost:8080", ""}, {"http://127.0.0.1:1", ""}, {"http://[::1]:8080", ""},
		{"https://staging.example.com", "staging.example.com"},
	}
	for _, c := range ok {
		if err := checkHost(c.base, c.allow); err != nil {
			t.Errorf("%v should pass: %v", c, err)
		}
	}
	bad := []struct{ base, allow string }{
		{"https://space-repetition-api.fly.dev", ""}, {"https://evil.com", "other.com"},
		{"http://localhost.evil.com", ""}, {"nonsense", ""},
	}
	for _, c := range bad {
		if checkHost(c.base, c.allow) == nil {
			t.Errorf("%v should be refused", c)
		}
	}
}

func TestPctAndToken(t *testing.T) {
	var d []time.Duration
	for i := 1; i <= 100; i++ {
		d = append(d, time.Duration(i)*time.Millisecond)
	}
	if got := pct(d, .95); got != 95*time.Millisecond {
		t.Errorf("p95 = %v", got)
	}
	if token(" https://x.dev/invite#abc \n") != "abc" || token("abc") != "abc" {
		t.Error("token parsing")
	}
}
