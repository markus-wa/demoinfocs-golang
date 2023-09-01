package sendtables2

import (
	"fmt"
	"regexp"
	"strconv"
)

var fieldTypeRe = regexp.MustCompile(`([^\<\[\*]+)(\<\s(.*)\s\>)?(\*)?(\[(.*)\])?`) // (\<\s.*?\s\>)?([.*?])?`)

type fieldType struct {
	baseType    string
	genericType *fieldType
	pointer     bool
	count       int
}

func newFieldType(name string) *fieldType {
	ss := fieldTypeRe.FindStringSubmatch(name)
	if len(ss) != 7 {
		panic(fmt.Sprintf("bad regexp: %s -> %#v", name, ss))
	}

	x := &fieldType{
		baseType: ss[1],
		pointer:  ss[4] == "*",
	}

	if ss[3] != "" {
		x.genericType = newFieldType(ss[3])
	}

	if n, ok := itemCounts[ss[6]]; ok {
		x.count = n
	} else if n, _ := strconv.Atoi(ss[6]); n > 0 {
		x.count = n
	} else if ss[6] != "" {
		x.count = 1024
	}

	return x
}

func (t *fieldType) String() string {
	x := t.baseType
	if t.genericType != nil {
		x += "<" + t.genericType.String() + ">"
	}
	if t.pointer {
		x += "*"
	}
	if t.count > 0 {
		x += "[" + strconv.Itoa(t.count) + "]"
	}
	return x
}
