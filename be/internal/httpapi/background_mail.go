package httpapi

import (
	"context"
	"sync"
	"time"
)

type backgroundMailExecutor struct {
	slots  chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	closed bool
}

func newBackgroundMailExecutor() *backgroundMailExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	return &backgroundMailExecutor{slots: make(chan struct{}, 8), ctx: ctx, cancel: cancel}
}

func (a *API) sendBackgroundMail(ctx context.Context, job func(context.Context)) {
	a.mail.send(ctx, job)
}

func (m *backgroundMailExecutor) send(ctx context.Context, job func(context.Context)) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.wg.Add(1)
	select {
	case m.slots <- struct{}{}:
		m.mu.Unlock()
		go func() {
			defer m.wg.Done()
			defer func() { <-m.slots }()
			mailCtx, cancel := context.WithTimeout(m.ctx, 45*time.Second)
			defer cancel()
			job(mailCtx)
		}()
	default:
		m.mu.Unlock()
		defer m.wg.Done()
		select {
		case m.slots <- struct{}{}:
			defer func() { <-m.slots }()
		case <-ctx.Done():
			return
		case <-m.ctx.Done():
			return
		}
		mailCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		stop := context.AfterFunc(m.ctx, cancel)
		defer stop()
		job(mailCtx)
	}
}

func (a *API) CloseBackgroundMail() {
	a.mail.close()
}

func (m *backgroundMailExecutor) close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	m.mu.Unlock()
	m.cancel()
	m.wg.Wait()
}
