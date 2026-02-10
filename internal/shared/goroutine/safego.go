// Package goroutine provides utilities for safely launching goroutines with panic recovery.
package goroutine

import (
	"fmt"
	"runtime/debug"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// SafeGo launches a goroutine with panic recovery. If the goroutine panics,
// the panic is caught and logged with stack trace instead of crashing the process.
func SafeGo(log logger.Interface, name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorw("goroutine panicked",
					"goroutine", name,
					"panic", fmt.Sprintf("%v", r),
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}
