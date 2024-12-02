package sendtables

// ServerClass is an auto-generated interface for property, intended to be used when mockability is needed.
// serverClass stores meta information about Entity types (e.g. palyers, teams etc.).
type ServerClass interface {
	// ID returns the server-class's ID.
	ID() int
	// Name returns the server-class's name.
	Name() string
	// DataTableID returns the data-table ID.
	DataTableID() int
	// DataTableName returns the data-table name.
	DataTableName() string
	// BaseClasses returns the base-classes of this server-class.
	BaseClasses() (res []ServerClass)
	// PropertyEntries returns the names of all property-entries on this server-class.
	PropertyEntries() []string
	// PropertyEntryDefinitions returns all property-entries on this server-class.
	PropertyEntryDefinitions() []PropertyEntry
	// OnEntityCreated registers a function to be called when a new entity is created from this serverClass.
	OnEntityCreated(handler EntityCreatedHandler)
	String() string
}
