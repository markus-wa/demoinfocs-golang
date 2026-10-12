package demoinfocs

import (
	"fmt"
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

// MatchInfoDecryptionKey extracts the net-message decryption key stored in `match730_*.dem.info`.
// Pass the whole contents of `match730_*.dem.info` to this function to get the key.
// See also: WithNetMessageDecryptionKey()
func MatchInfoDecryptionKey(b []byte) ([]byte, error) {
	m := new(msg.CDataGCCStrike15V2_MatchInfo)

	err := proto.Unmarshal(b, m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MatchInfo message: %w", err)
	}

	k := []byte(strings.ToUpper(fmt.Sprintf("%016x", m.GetWatchablematchinfo().GetClDecryptdataKeyPub())))

	return k, nil
}
