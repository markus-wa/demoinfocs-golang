// DO NOT EDIT: Auto generated

package demoinfocs

import (
	"github.com/markus-wa/demoinfocs-golang/common"
)

// IParticipants is an auto-generated interface for Participants.
// Participants provides helper functions on top of the currently connected players.
// E.g. ByUserID(), ByEntityID(), TeamMembers(), etc.
//
// See GameState.Participants()
type IParticipants interface {
	// ByUserID returns all currently connected players in a map where the key is the user-ID.
	// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
	// Includes spectators.
	ByUserID() map[int]*common.Player
	// ByEntityID returns all currently connected players in a map where the key is the entity-ID.
	// The returned map is a snapshot and is not updated on changes (not a reference to the actual, underlying map).
	// Includes spectators.
	ByEntityID() map[int]*common.Player
	// All returns all currently known players & spectators, including disconnected ones, of the demo.
	// The returned slice is a snapshot and is not updated on changes.
	All() []*common.Player
	// Connected returns all currently connected players & spectators.
	// The returned slice is a snapshot and is not updated on changes.
	Connected() []*common.Player
	// Playing returns all players that aren't spectating or unassigned.
	// The returned slice is a snapshot and is not updated on changes.
	Playing() []*common.Player
	// TeamMembers returns all players belonging to the requested team at this time.
	// The returned slice is a snapshot and is not updated on changes.
	TeamMembers(team common.Team) []*common.Player
	// FindByHandle attempts to find a player by his entity-handle.
	// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
	//
	// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
	FindByHandle(handle int) *common.Player
	// SpottersOf returns a list of all players who have spotted the passed player.
	SpottersOf(spotted *common.Player) (spotters []*common.Player)
	// SpottedBy returns a list of all players that the passed player has spotted.
	SpottedBy(spotter *common.Player) (spotted []*common.Player)
}
