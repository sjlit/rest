// Package safelog provides panic-recovered logging helpers used by hook
// execution paths where a panic in user code must not abort the main flow
// but still needs to be visible to operators.
package safelog

import (
	"log"
	"runtime/debug"
)

// SafeRun executes fn and converts any panic into a logged error instead of
// propagating. Use in after-hook runners where the parent operation has
// already succeeded and a user callback must not corrupt the result.
func SafeRun(phase string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[rest] %s hook panic recovered: %v\n%s", phase, r, debug.Stack())
		}
	}()
	fn()
}
