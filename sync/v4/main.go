package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var rwMutex sync.RWMutex
	ch := make(chan int) // unbuffered channel

	// goroutine 1：hold write lock，send msg to channel
	go func() {
		rwMutex.Lock()
		fmt.Println("goroutine1：獲取寫鎖，準備發送到通道")
		ch <- 123 // 阻塞等待接收者
		fmt.Println("goroutine1：發送完成，釋放寫鎖")
		rwMutex.Unlock()
	}()

	time.Sleep(100 * time.Millisecond) // 確保goroutine1先執行

	// goroutine2：try to octain read lock, and receive msg from channel
	go func() {
		fmt.Println("goroutine2：嘗試獲取讀鎖")
		rwMutex.RLock() // 阻塞，因為協程1持有寫鎖
		fmt.Println("goroutine2：獲取到讀鎖，準備從通道接收")
		x := <-ch
		fmt.Println("goroutine2：讀到", x)
		rwMutex.RUnlock()
	}()

	time.Sleep(20 * time.Second)
	fmt.Println("結束")
}
