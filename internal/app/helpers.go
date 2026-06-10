package app

import (
	"fmt"
	"llm-util/constant"
	"time"
)

// 显示动态 Loading 动画
func ShowLoading(done chan struct{}) {
	frames := []string{"-", "\\", "|", "/"}
	i := 0

	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("\r"+constant.Yellow+"%s"+constant.Reset, frames[i%len(frames)])
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}
