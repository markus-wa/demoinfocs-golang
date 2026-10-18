package common

import (
	"fmt"

	st "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/sendtables"
)

// cachedProp indexes the CS2 properties the Player getters read through cachedProps.
type cachedProp int

const (
	// pawn properties
	propHealth cachedProp = iota
	propArmor
	propLifeState
	propFlags
	propGroundEntity
	propIsScoped
	propIsWalking
	propHasHelmet
	propHasDefuser
	propIsGrabbingHostage
	propEyeAngles
	propActiveWeapon
	propFlashbangAmmo
	propVelocityX
	propVelocityY
	propVelocityZ

	// controller properties
	propMoney
	propPing

	cachedPropCount
)

var cachedPropNames = [cachedPropCount]string{
	propHealth:            "m_iHealth",
	propArmor:             "m_ArmorValue",
	propLifeState:         "m_lifeState",
	propFlags:             "m_fFlags",
	propGroundEntity:      "m_hGroundEntity",
	propIsScoped:          "m_bIsScoped",
	propIsWalking:         "m_bIsWalking",
	propHasHelmet:         "m_pItemServices.m_bHasHelmet",
	propHasDefuser:        "m_pItemServices.m_bHasDefuser",
	propIsGrabbingHostage: "m_bIsGrabbingHostage",
	propEyeAngles:         "m_angEyeAngles",
	propActiveWeapon:      "m_pWeaponServices.m_hActiveWeapon",
	propFlashbangAmmo:     "m_pWeaponServices.m_iAmmo.0014",
	propVelocityX:         "m_vecVelocity.m_vecX",
	propVelocityY:         "m_vecVelocity.m_vecY",
	propVelocityZ:         "m_vecVelocity.m_vecZ",
	propMoney:             "m_pInGameMoneyServices.m_iAccount",
	propPing:              "m_iPing",
}

// cachedProps caches an entity's property handles so that the getters don't look them up by name on
// every call (most of them are called for every player on every frame by consumers tracking positions).
// It's keyed by the entity the handles were resolved on: a re-created pawn / controller resets it.
// Only existing properties are cached, a property the entity doesn't have (yet) is looked up again on
// the next call - the entity's own lookup handles that case cheaply and may start resolving it later.
type cachedProps struct {
	entity st.Entity
	props  [cachedPropCount]st.Property
}

// value returns the entity's property idx, false if the entity doesn't have it.
func (c *cachedProps) value(entity st.Entity, idx cachedProp) (st.PropertyValue, bool) {
	if c.entity != entity {
		*c = cachedProps{entity: entity}
	}

	prop := c.props[idx]
	if prop == nil {
		prop = entity.Property(cachedPropNames[idx])
		if prop == nil {
			return st.PropertyValue{S2: true}, false
		}

		c.props[idx] = prop
	}

	return prop.Value(), true
}

// mustValue is value() panicking if the entity doesn't have the property, like Entity.PropertyValueMust().
func (c *cachedProps) mustValue(entity st.Entity, idx cachedProp) st.PropertyValue {
	v, ok := c.value(entity, idx)
	if !ok {
		panic(fmt.Sprintf("property '%s' not found", cachedPropNames[idx]))
	}

	return v
}

// pawnValue returns the pawn's property idx, false if the player has no pawn or the pawn doesn't have it.
func (p *Player) pawnValue(idx cachedProp) (st.PropertyValue, bool) {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return st.PropertyValue{S2: true}, false
	}

	return p.pawnProps.value(pawn, idx)
}

// pawnValueMust returns the pawn's property idx, false if the player has no pawn.
// Like the getXXX helpers it panics if the pawn doesn't have the property.
func (p *Player) pawnValueMust(idx cachedProp) (st.PropertyValue, bool) {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return st.PropertyValue{S2: true}, false
	}

	return p.pawnProps.mustValue(pawn, idx), true
}

// pawnInt is getInt(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnInt(idx cachedProp) int {
	v, ok := p.pawnValueMust(idx)
	if !ok {
		return 0
	}

	return v.Int()
}

// pawnUInt64 is getUInt64(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnUInt64(idx cachedProp) uint64 {
	v, ok := p.pawnValueMust(idx)
	if !ok {
		return 0
	}

	return v.S2UInt64()
}

// pawnBool is getBool(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnBool(idx cachedProp) bool {
	v, ok := p.pawnValueMust(idx)
	if !ok {
		return false
	}

	return v.BoolVal()
}

// pawnFloatIfExists is getFloatIfExists(pawn, ...) through the property cache, for a pawn already resolved.
func (p *Player) pawnFloatIfExists(pawn st.Entity, idx cachedProp) (float32, bool) {
	v, ok := p.pawnProps.value(pawn, idx)
	if !ok || v.Any == nil {
		return 0, false
	}

	floatVal, ok := v.Any.(float32)
	if !ok {
		return 0, false
	}

	return floatVal, true
}

// controllerValueMust returns the controller's property idx, false if the player has no controller entity.
// Like the getXXX helpers it panics if the controller doesn't have the property.
func (p *Player) controllerValueMust(idx cachedProp) (st.PropertyValue, bool) {
	if p.Entity == nil {
		return st.PropertyValue{S2: true}, false
	}

	return p.controllerProps.mustValue(p.Entity, idx), true
}
