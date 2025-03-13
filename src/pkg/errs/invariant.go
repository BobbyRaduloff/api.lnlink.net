//coverage:ignore file
package errs

import "fmt"

func Invariant(condition bool, message string) {
	if !condition {
		panic(fmt.Sprintf("Invariant failed: %s", message))
	}
}
