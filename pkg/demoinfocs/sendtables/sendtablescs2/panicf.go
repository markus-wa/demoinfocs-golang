package sendtablescs2

import "fmt"

func _panicf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}
