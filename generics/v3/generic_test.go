package v3

import "testing"

func TestAssertFunctions(t *testing.T) {
	t.Run("asserting on integers", func(t *testing.T) {
		AssertEqual(t, 1, 1)

		AssertNotEqual(t, 1, 2)
	})

	t.Run("asserting on strings", func(t *testing.T) {
		AssertEqual(t, "hello", "hello")

		AssertNotEqual(t, "hello", "Grace")
	})
	// AssertEqual(t, 1, "1") // uncomment to see the error
}

// [T comparable]︰類型參數的類型是 comparable，我們給它的標籤是 T
// 我們使用 comparable 因為我們要向 Compiler 描述，
// 我們希望在函式中對 T 類型的東西使用 == 和 != 運算符號，我們想要比較！
// 如果您嘗試將類型變更為 any，func AssertEqual[T any](t *testing.T, got, want T)
// 會出現 prog.go2:15:5: cannot compare got != want (operator != not defined for T)
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	// 因為這裡有用到 != ，這operator被定義在comparable，而 any 卻沒定義這operator
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func AssertNotEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()

	if got == want {
		t.Errorf("didn't want %v", got)
	}
}
