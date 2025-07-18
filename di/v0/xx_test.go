package v0_test

import (
	"fmt"
	"testing"
	"time"
)

// SUT
// 使用方法入參注入時間
func GreetWithTime(name string, currentTime time.Time) string {
	if currentTime.Hour() < 12 {
		return fmt.Sprintf("早安, %s", name)
	} else if currentTime.Hour() < 18 {
		return fmt.Sprintf("午安, %s", name)
	}
	return fmt.Sprintf("晚安, %s", name)
}

// Test case
func TestGreetWithTime(t *testing.T) {
	// 測試早上問候
	t.Run("morning greeting", func(t *testing.T) {
		morningTime := time.Date(2023, time.January, 1, 8, 0, 0, 0, time.UTC)
		result := GreetWithTime("Chris", morningTime)
		expected := "早安, Chris"
		if result != expected {
			t.Errorf("Expected %q but got %q", expected, result)
		}
	})

	// 測試下午問候
	t.Run("afternoon greeting", func(t *testing.T) {
		afternoonTime := time.Date(2023, time.January, 1, 15, 0, 0, 0, time.UTC)
		result := GreetWithTime("Chris", afternoonTime)
		expected := "午安, Chris"
		if result != expected {
			t.Errorf("Expected %q but got %q", expected, result)
		}
	})

	// 測試晚上問候
	t.Run("evening greeting", func(t *testing.T) {
		eveningTime := time.Date(2023, time.January, 1, 20, 0, 0, 0, time.UTC)
		result := GreetWithTime("Chris", eveningTime)
		expected := "晚安, Chris"
		if result != expected {
			t.Errorf("Expected %q but got %q", expected, result)
		}
	})
}
