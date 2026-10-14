// Package stbridge gives internal/stfake access to the send-tables parser's synthetic entities.
//
// The send-tables package sets the hooks in its init(). They are typed with any because the
// send-tables package imports this package, so this package can't import it back.
// Use internal/stfake instead of calling the hooks directly.
package stbridge

var (
	// NewParser returns a new *Parser that knows server-classes with the given names.
	NewParser func(classes []string) any

	// NewEntity returns a new synthetic *Entity that declares exactly the given properties, in that (field-path) order.
	// parser is a *Parser and may be nil; if it's not, class must be one of its server-classes.
	// The entity doesn't keep a reference to the parser, pass the same one to Create, Set and Destroy.
	NewEntity func(parser any, class string, id, serial int, names []string, values []any) any

	// Create fires everything an entity packet fires when it creates the entity.
	// parser is the entity's *Parser (may be nil).
	Create func(entity, parser any)

	// Set sets the given property values of the entity, in field-path order.
	// If fire is set, each property's update handlers fire right after it's set.
	// If finish is set as well, the entity's update is finished like at the end of an entity packet.
	// parser is the entity's *Parser (may be nil).
	Set func(entity, parser any, names []string, values []any, fire, finish bool)

	// Destroy deletes the entity like an entity packet does.
	// parser is the entity's *Parser (may be nil).
	Destroy func(entity, parser any)
)
