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
	// The map is a snapshot and is not updated (not a reference to the actual, underlying map).
	// Includes spectators.
	ByUserID() map[int]*common.Player
	// ByEntityID returns all currently connected players in a map where the key is the entity-ID.
	// The map is a snapshot and is not updated (not a reference to the actual, underlying map).
	// Includes spectators.
	ByEntityID() map[int]*common.Player
	// All returns all currently connected players & spectators.
	All() []*common.Player
	// Playing returns all players that aren't spectating or unassigned.
	Playing() []*common.Player
	// TeamMembers returns all players belonging to the requested team at this time.
	TeamMembers(team common.Team) []*common.Player
	// FindByHandle attempts to find a player by his entity-handle.
	// The entity-handle is often used in entity-properties when referencing other entities such as a weapon's owner.
	//
	// Returns nil if not found or if handle == invalidEntityHandle (used when referencing no entity).
	FindByHandle(handle int) *common.Player
}
