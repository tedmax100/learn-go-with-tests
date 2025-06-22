package v5

import (
	"sync"
)

// Counter 是一個併發安全的計數器
type Counter struct {
	mu    sync.Mutex
	value int
}

// Inc 增加計數器的值
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Value 返回計數器的當前值
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}
