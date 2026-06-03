package safelog

import (
	"context"
	"log"
	"strings"
	"testing"
)

func TestSafeRunRecoversPanic(t *testing.T) {
	// Capture log output to verify it gets called
	var got []byte
	old := log.Writer()
	defer log.SetOutput(old)
	log.SetOutput(stringWriterFunc(func(p []byte) (int, error) {
		got = p
		return len(p), nil
	}))

	called := false
	SafeRun(context.Background(), "test", func() {
		called = true
		panic("boom")
	})
	if !called {
		t.Error("fn should have been called before panic")
	}
	if !strings.Contains(string(got), "test hook panic recovered") {
		t.Errorf("expected log to mention test phase, got: %q", string(got))
	}
	if !strings.Contains(string(got), "boom") {
		t.Errorf("expected log to include panic value, got: %q", string(got))
	}
}

func TestSafeRunNoPanic(t *testing.T) {
	called := false
	SafeRun(context.Background(), "ok", func() {
		called = true
	})
	if !called {
		t.Error("fn should have been called")
	}
}

type stringWriterFunc func([]byte) (int, error)

func (f stringWriterFunc) Write(p []byte) (int, error) { return f(p) }
