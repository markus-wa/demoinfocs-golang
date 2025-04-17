package sendtablescs2

import "fmt"

func _panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
