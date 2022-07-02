package context

type Session interface {

	// Get
	Get(name string, value ...interface{}) interface{}

	// Set
	Set(name string, value interface{})

	// All
	All() map[string]interface{}

	// Remove
	Remove(name string) interface{}

	// Forget
	Forget(names ...string)

	// Clear
	Clear()

	// Save
	Save()
}
