package monitor

import (
	"context"
	"time"
)

// TokenMonitor : 簡化版本
type TokenMonitor struct {
	notificationChan    <-chan string
	ticker              *time.Ticker
	checkFunc           func(context.Context)
	interval            time.Duration
	ctx                 context.Context
	cancel              context.CancelFunc
	ProcessNotification func(string)
}

// NewTokenMonitor: constructor
func NewTokenMonitor(notificationChan <-chan string) *TokenMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &TokenMonitor{
		notificationChan: notificationChan,
		interval:         1 * time.Second,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// SetCheckFunc : set check function
func (tm *TokenMonitor) SetCheckFunc(fn func(context.Context)) {
	tm.checkFunc = fn
}

// SetInterval : set scan interval
func (tm *TokenMonitor) SetInterval(interval time.Duration) {
	tm.interval = interval
	if tm.ticker != nil {
		tm.ticker.Reset(interval)
	}
}

// Run : 啟動 monitor instance
func (tm *TokenMonitor) Run() {
	tm.ticker = time.NewTicker(tm.interval)

	for {
		select {
		case msg, ok := <-tm.notificationChan:
			if !ok {
				return // since channel is closed and then return the process
			}
			go tm.ProcessNotification(msg)

		case <-tm.ticker.C:
			if tm.checkFunc != nil {
				go tm.checkFunc(tm.ctx)
			}

		case <-tm.ctx.Done():
			return // since context is cancled and then return
		}
	}
}

// Stop : stop monitor
func (tm *TokenMonitor) Stop() {
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
	tm.cancel()
}
