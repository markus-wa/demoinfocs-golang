package demoinfocs

import (
	"fmt"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// MatchInfoDecryptionKey extracts the net-message decryption key stored in `match730_*.dem.info`.
// Pass the whole contents of `match730_*.dem.info` to this function to get the key.
// See also: ParserConfig.NetMessageDecryptionKey
func MatchInfoDecryptionKey(b []byte) ([]byte, error) {
	m := new(msg.CDataGCCStrike15V2_MatchInfo)

	err := proto.Unmarshal(b, m)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal MatchInfo message")
	}

	k := []byte(strings.ToUpper(fmt.Sprintf("%016x", m.GetWatchablematchinfo().GetClDecryptdataKeyPub())))

	return k, nil
}
