package migrations

import (
	"database/sql"

	"github.com/glugox/unogo/db/migration"
)

func init() {
	migration.AddMigration(upcrateUsersTable, downcrateUsersTable)
}
func upcrateUsersTable(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	tx.Exec("CREATE TABLE users")
	return nil
}
func downcrateUsersTable(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	tx.Exec("DROP TABLE users")
	return nil
}
