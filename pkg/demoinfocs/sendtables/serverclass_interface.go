package sendtables

// ServerClass is an auto-generated interface for property, intended to be used when mockability is needed.
// serverClass stores meta information about Entity types (e.g. palyers, teams etc.).
type ServerClass interface {
	// ID returns the server-class's ID.
	ID() int
	// Name returns the server-class's name.
	Name() string
	// PropertyEntries returns the names of all property-entries on this server-class.
	PropertyEntries() []string
	// OnEntityCreated registers a function to be called when a new entity is created from this serverClass.
	OnEntityCreated(handler EntityCreatedHandler)
	String() string
}
