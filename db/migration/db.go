package migration

import (
	"database/sql"
	"fmt"
)

// OpenDBWithDriver Creates a connection to a database
func OpenDBWithDriver(driver string, dbstring string) (*sql.DB, error) {
	if err := SetDialect(driver); err != nil {
		return nil, err
	}

	switch driver {
	case "mssql":
		driver = "sqlserver"
	case "redshift":
		driver = "postgres"
	case "tidb":
		driver = "mysql"
	}

	switch driver {
	case "postgres", "pgx", "sqlite3", "sqlite", "mysql", "sqlserver", "clickhouse":
		return sql.Open(driver, dbstring)
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
