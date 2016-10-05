package dt

import (
	"math"
)

type Parser struct {
	sendTables         []*SendTable
	serverClasses      []*ServerClass
	currentExcludes    []*ExcludeEntry
	currentBaseclasses []*ServerClass
}

func (p Parser) ClassBits() int {
	return int(math.Ceil(math.Log2(float64(len(p.serverClasses)))))
}

func (p Parser) ParsePacket() int {
	return 1
}
