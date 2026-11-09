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
	props  [cachedPropCount]rawValuer
}

// rawValuer is implemented by the CS2 properties: RawValue() is Value().Any.
// The cache reads raw values because st.PropertyValue is a large struct, copying it through every call of the
// getters is noticeable for consumers calling them for every player on every frame.
type rawValuer interface {
	RawValue() any
}

// propertyRawValuer adapts properties that aren't rawValuers (e.g. mocks). The cached properties are CS2
// properties, so their Value() is st.PropertyValue{Any: Value().Any, S2: true}.
type propertyRawValuer struct {
	st.Property
}

func (p propertyRawValuer) RawValue() any {
	return p.Value().Any
}

// s2Value wraps a raw value as the st.PropertyValue a CS2 property's Value() returns.
func s2Value(v any) st.PropertyValue {
	return st.PropertyValue{Any: v, S2: true}
}

// value returns the raw value of the entity's property idx, false if the entity doesn't have it.
func (c *cachedProps) value(entity st.Entity, idx cachedProp) (any, bool) {
	if c.entity != entity {
		*c = cachedProps{entity: entity}
	}

	prop := c.props[idx]
	if prop == nil {
		property := entity.Property(cachedPropNames[idx])
		if property == nil {
			return nil, false
		}

		var ok bool
		prop, ok = property.(rawValuer)
		if !ok {
			prop = propertyRawValuer{property}
		}

		c.props[idx] = prop
	}

	return prop.RawValue(), true
}

// mustValue is value() panicking if the entity doesn't have the property, like Entity.PropertyValueMust().
func (c *cachedProps) mustValue(entity st.Entity, idx cachedProp) any {
	v, ok := c.value(entity, idx)
	if !ok {
		panic(fmt.Sprintf("property '%s' not found", cachedPropNames[idx]))
	}

	return v
}

// pawnRawMust returns the raw value of the pawn's property idx, false if the player has no pawn.
// Like the getXXX helpers it panics if the pawn doesn't have the property.
func (p *Player) pawnRawMust(idx cachedProp) (any, bool) {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return nil, false
	}

	return p.pawnProps.mustValue(pawn, idx), true
}

// pawnValue returns the pawn's property idx, false if the player has no pawn or the pawn doesn't have it.
func (p *Player) pawnValue(idx cachedProp) (st.PropertyValue, bool) {
	pawn := p.PlayerPawnEntity()
	if pawn == nil {
		return s2Value(nil), false
	}

	v, ok := p.pawnProps.value(pawn, idx)

	return s2Value(v), ok
}

// pawnValueMust returns the pawn's property idx, false if the player has no pawn.
// Like the getXXX helpers it panics if the pawn doesn't have the property.
func (p *Player) pawnValueMust(idx cachedProp) (st.PropertyValue, bool) {
	v, ok := p.pawnRawMust(idx)

	return s2Value(v), ok
}

// pawnInt is getInt(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnInt(idx cachedProp) int {
	v, ok := p.pawnRawMust(idx)
	if !ok {
		return 0
	}

	return s2Value(v).Int()
}

// pawnUInt64 is getUInt64(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnUInt64(idx cachedProp) uint64 {
	v, ok := p.pawnRawMust(idx)
	if !ok {
		return 0
	}

	return s2Value(v).S2UInt64()
}

// pawnBool is getBool(p.PlayerPawnEntity(), ...) through the property cache.
func (p *Player) pawnBool(idx cachedProp) bool {
	v, ok := p.pawnRawMust(idx)
	if !ok {
		return false
	}

	return s2Value(v).BoolVal()
}

// pawnFloatIfExists is getFloatIfExists(pawn, ...) through the property cache, for a pawn already resolved.
func (p *Player) pawnFloatIfExists(pawn st.Entity, idx cachedProp) (float32, bool) {
	v, ok := p.pawnProps.value(pawn, idx)
	if !ok || v == nil {
		return 0, false
	}

	floatVal, ok := v.(float32)
	if !ok {
		return 0, false
	}

	return floatVal, true
}

// controllerValueMust returns the controller's property idx, false if the player has no controller entity.
// Like the getXXX helpers it panics if the controller doesn't have the property.
func (p *Player) controllerValueMust(idx cachedProp) (st.PropertyValue, bool) {
	if p.Entity == nil {
		return s2Value(nil), false
	}

	return s2Value(p.controllerProps.mustValue(p.Entity, idx)), true
}
