package v2

import (
	"sync"
	"testing"
)

func TestCounter(t *testing.T) {

	t.Run("incrementing the counter 3 times leaves it at 3", func(t *testing.T) {
		counter := &Counter{}
		counter.Inc()
		counter.Inc()
		counter.Inc()

		assertCounter(t, counter, 3)
	})

	t.Run("it runs safely concurrently", func(t *testing.T) {
		wantedCount := 1000
		counter := &Counter{}

		var wg sync.WaitGroup
		wg.Add(wantedCount)

		for i := 0; i < wantedCount; i++ {
			go func() {
				counter.Inc()
				wg.Done()
			}()
		}
		wg.Wait()

		assertCounter(t, counter, wantedCount)
	})

	t.Run("it runs safely concurrently by Atomic", func(t *testing.T) {
		wantedCount := 1000
		counter := &AtomicCounter{}

		var wg sync.WaitGroup
		wg.Add(wantedCount)

		for i := 0; i < wantedCount; i++ {
			go func() {
				counter.Inc()
				wg.Done()
			}()
		}
		wg.Wait()

		assertCounter(t, counter, wantedCount)
	})

}

func assertCounter(t testing.TB, got ICounter, want int) {
	t.Helper()
	if got.Value() != int64(want) {
		t.Errorf("got %d, want %d", got.Value(), want)
	}
}
