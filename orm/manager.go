package orm

import (
	"fmt"
)

type Config struct {
	Driver string
	Prefix string
	Dsn    string
}

// Database manager.
type Manager struct {
	configs     map[string]Config
	connections map[string]Connection
}

// Connect Get a database connection instance.
func (m *Manager) Connect(name string) (*Connection, error) {
	if conn, ok := m.connections[name]; ok {
		return &conn, nil
	}

	return m.make(name)
}

func (m *Manager) make(name string) (*Connection, error) {
	config, ok := m.configs[name]
	if !ok {
		return nil, fmt.Errorf("database [%s] not configured", name)
	}

	return Open(config)
}
