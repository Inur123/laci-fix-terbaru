package httpapi

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"laci-backend/internal/config"
)

func TestBackgroundMailBackpressureAndShutdown(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	started := make(chan struct{}, 9)
	release := make(chan struct{})
	for i := 0; i < cap(a.mail.slots); i++ {
		a.sendBackgroundMail(context.Background(), func(context.Context) {
			started <- struct{}{}
			<-release
		})
	}
	for i := 0; i < cap(a.mail.slots); i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("background mail did not start")
		}
	}
	blockedDone := make(chan struct{})
	go func() {
		a.sendBackgroundMail(context.Background(), func(context.Context) {
			started <- struct{}{}
		})
		close(blockedDone)
	}()
	select {
	case <-started:
		t.Fatal("more than eight email jobs ran concurrently")
	case <-blockedDone:
		t.Fatal("saturated email job did not apply backpressure")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-blockedDone:
	case <-time.After(time.Second):
		t.Fatal("email job stayed blocked after capacity became available")
	}
	a.CloseBackgroundMail()
	if len(a.mail.slots) != 0 {
		t.Fatalf("email slots leaked: %d", len(a.mail.slots))
	}
	var ran atomic.Bool
	a.sendBackgroundMail(context.Background(), func(context.Context) { ran.Store(true) })
	if ran.Load() {
		t.Fatal("email task started after shutdown")
	}
}

func TestBackgroundMailShutdownCancelsActiveWork(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	started := make(chan struct{})
	a.sendBackgroundMail(context.Background(), func(ctx context.Context) {
		close(started)
		<-ctx.Done()
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("email job did not start")
	}
	done := make(chan struct{})
	go func() {
		a.CloseBackgroundMail()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("email shutdown did not cancel active work")
	}
}
