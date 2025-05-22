package main_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestTimerRaceCondition
func TestTimerRaceCondition(t *testing.T) {
	start := time.Now()
	var timer *time.Timer

	timer = time.AfterFunc(randomDuration(), func() {
		fmt.Println(time.Now().Sub(start))
		timer.Reset(randomDuration())
	})

	time.Sleep(1 * time.Second)
}

// TestTimerWithMutex
func TestTimerWithMutex(t *testing.T) {
	start := time.Now()
	var timer *time.Timer
	var mu sync.Mutex

	mu.Lock()
	timer = time.AfterFunc(randomDuration(), func() {
		mu.Lock()
		fmt.Println(time.Now().Sub(start))
		timer.Reset(randomDuration())
		mu.Unlock()
	})
	mu.Unlock()

	time.Sleep(1 * time.Second)
}

// TestTimerWithRecursiveFunc
func TestTimerWithRecursiveFunc(t *testing.T) {
	start := time.Now()

	var scheduleNext func()
	scheduleNext = func() {
		fmt.Println(time.Now().Sub(start))
		time.AfterFunc(randomDuration(), scheduleNext)
	}

	time.AfterFunc(randomDuration(), scheduleNext)

	time.Sleep(1 * time.Second)
}

func randomDuration() time.Duration {
	return time.Duration(rand.Int63n(1e9))
}
