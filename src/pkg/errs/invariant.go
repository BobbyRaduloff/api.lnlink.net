//coverage:ignore file
package errs

import "fmt"

func Invariant(condition bool, message string, args ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("Invariant failed: "+message, args...))
	}
}
