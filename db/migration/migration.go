package migration

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/glugox/unogo/log"
	"github.com/glugox/unogo/support/os/filesystem"
)

var (
	minVersion             = int64(0)
	maxVersion             = int64((1 << 63) - 1)
	timestampFormat        = "20220703000000"
	verbose                = true
	verTplSeq              = "%05v"
	matchSQLComments       = regexp.MustCompile(`(?m)^--.*$[\r\n]*`)
	matchEmptyEOL          = regexp.MustCompile(`(?m)^$[\r\n]*`)
	tableName              = "migrations"
	baseFS           fs.FS = filesystem.OsFS{}
)

// MigrationRecord struct.
type MigrationRecord struct {
	VersionID int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

// Migration struct.
type Migration struct {
	Version      int64
	Next         int64  // next version, or -1 if none
	Previous     int64  // previous version, -1 if none
	Source       string // path to .sql script or go file
	Registered   bool
	UpFn         func(*sql.Tx) error // Up go migration function
	DownFn       func(*sql.Tx) error // Down go migration function
	noVersioning bool
}

// SetVerbose set the goose verbosity mode
func SetVerbose(v bool) {
	verbose = v
}

// TableName returns migration db table name
func TableName() string {
	return tableName
}

// SetTableName set migration db table name
func SetTableName(n string) {
	tableName = n
}

func (m *Migration) String() string {
	return fmt.Sprintf(m.Source)
}

// Up runs an up migration.
func (m *Migration) Up(db *sql.DB) error {
	if err := m.run(db, true); err != nil {
		return err
	}
	return nil
}

// Down runs a down migration.
func (m *Migration) Down(db *sql.DB) error {
	if err := m.run(db, false); err != nil {
		return err
	}
	return nil
}

func (m *Migration) run(db *sql.DB, direction bool) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := baseFS.Open(m.Source)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to open SQL migration file: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := parseSQLMigration(f, direction)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to parse SQL migration file: %w", filepath.Base(m.Source), err)
		}

		if err := runSQLMigration(db, statements, useTx, m.Version, direction, m.noVersioning); err != nil {
			return fmt.Errorf("ERROR %v: failed to run SQL migration: %w", filepath.Base(m.Source), err)
		}

		if len(statements) > 0 {
			log.Info("OK   ", filepath.Base(m.Source))
		} else {
			log.Info("EMPTY", filepath.Base(m.Source))
		}

	case ".go":
		if !m.Registered {
			return fmt.Errorf("ERROR %v: failed to run Go migration: Go functions must be registered and built into a custom binary", m.Source)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("ERROR failed to begin transaction: %w", err)
		}

		fn := m.UpFn
		if !direction {
			fn = m.DownFn
		}

		if fn != nil {
			// Run Go migration function.
			if err := fn(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("ERROR %v: failed to run Go migration function %T: %w", filepath.Base(m.Source), fn, err)
			}
		}
		if !m.noVersioning {
			if direction {
				if _, err := tx.Exec(GetDialect().insertVersionSQL(), m.Version, direction); err != nil {
					tx.Rollback()
					return fmt.Errorf("ERROR failed to execute transaction: %w", err)
				}
			} else {
				if _, err := tx.Exec(GetDialect().deleteVersionSQL(), m.Version); err != nil {
					tx.Rollback()
					return fmt.Errorf("ERROR failed to execute transaction: %w", err)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("ERROR failed to commit transaction: %w", err)
		}

		if fn != nil {
			log.Debug("OK   ", filepath.Base(m.Source))
		} else {
			log.Debug("EMPTY", filepath.Base(m.Source))
		}

		return nil
	}

	return nil
}

// NumericComponent looks for migration scripts with names in the form:
// XXX_descriptivename.ext where XXX specifies the version number
// and ext specifies the type of migration
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}

// Version prints the current version of the database.
func Version(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	if option.noVersioning {
		var current int64
		migrations, err := CollectMigrations(dir, minVersion, maxVersion)
		if err != nil {
			return fmt.Errorf("failed to collect migrations: %s", err)
		}
		if len(migrations) > 0 {
			current = migrations[len(migrations)-1].Version
		}
		log.Info("migration: file version %v\n", current)
		return nil
	}

	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}
	log.Info("migration: version %v\n", current)
	return nil
}

// Run runs a migration command.
func Run(command string, db *sql.DB, dir string, args ...string) error {
	return run(command, db, dir, args)
}

// Run runs a migration command with options.
func RunWithOptions(command string, db *sql.DB, dir string, args []string, options ...OptionsFunc) error {
	return run(command, db, dir, args, options...)
}

func run(command string, db *sql.DB, dir string, args []string, options ...OptionsFunc) error {
	switch command {
	case "up":
		if err := Up(db, dir, options...); err != nil {
			return err
		}
	case "up-by-one":
		if err := UpByOne(db, dir, options...); err != nil {
			return err
		}
	case "up-to":
		if len(args) == 0 {
			return fmt.Errorf("up-to must be of form: migration [OPTIONS] DRIVER DBSTRING up-to VERSION")
		}

		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := UpTo(db, dir, version, options...); err != nil {
			return err
		}
	case "create":
		if len(args) == 0 {
			return fmt.Errorf("create must be of form: migration [OPTIONS] DRIVER DBSTRING create NAME [go|sql]")
		}

		migrationType := "go"
		if len(args) == 2 {
			migrationType = args[1]
		}
		if err := Create(db, dir, args[0], migrationType); err != nil {
			return err
		}
	case "down":
		if err := Down(db, dir, options...); err != nil {
			return err
		}
	case "down-to":
		if len(args) == 0 {
			return fmt.Errorf("down-to must be of form: migration [OPTIONS] DRIVER DBSTRING down-to VERSION")
		}

		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := DownTo(db, dir, version, options...); err != nil {
			return err
		}
	case "fix":
		return fmt.Errorf("fix is not implemented")
		/*if err := Fix(dir); err != nil {
			return err
		}*/
	case "redo":
		/*if err := Redo(db, dir, options...); err != nil {
			return err
		}*/
		return fmt.Errorf("redo is not implemented")
	case "reset":
		/*if err := Reset(db, dir, options...); err != nil {
			return err
		}*/
		return fmt.Errorf("reset is not implemented")
	case "status":
		/*if err := Status(db, dir, options...); err != nil {
			return err
		}*/
		return fmt.Errorf("status is not implemented")
	case "version":
		if err := Version(db, dir, options...); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}

func runSQLMigration(db *sql.DB, statements []string, useTx bool, v int64, direction bool, noVersioning bool) error {
	if useTx {
		// TRANSACTION.

		log.Debug("Begin transaction")

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		for _, query := range statements {
			log.Debug("Executing statement: %s\n", clearStatement(query))
			if err = execQuery(tx.Exec, query); err != nil {
				log.Debug("Rollback transaction")
				tx.Rollback()
				return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
			}
		}

		if !noVersioning {
			if direction {
				if err := execQuery(tx.Exec, GetDialect().insertVersionSQL(), v, direction); err != nil {
					log.Debug("Rollback transaction")
					tx.Rollback()
					return fmt.Errorf("failed to insert new migration version: %w", err)
				}
			} else {
				if err := execQuery(tx.Exec, GetDialect().deleteVersionSQL(), v); err != nil {
					log.Debug("Rollback transaction")
					tx.Rollback()
					return fmt.Errorf("failed to delete migration version: %w", err)
				}
			}
		}

		log.Debug("Commit transaction")
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	// NO TRANSACTION.
	for _, query := range statements {
		log.Debug("Executing statement: %s", clearStatement(query))
		if err := execQuery(db.Exec, query); err != nil {
			return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
		}
	}
	if !noVersioning {
		if direction {
			if err := execQuery(db.Exec, GetDialect().insertVersionSQL(), v, direction); err != nil {
				return fmt.Errorf("failed to insert new migration version: %w", err)
			}
		} else {
			if err := execQuery(db.Exec, GetDialect().deleteVersionSQL(), v); err != nil {
				return fmt.Errorf("failed to delete migration version: %w", err)
			}
		}
	}

	return nil
}

func execQuery(fn func(string, ...interface{}) (sql.Result, error), query string, args ...interface{}) error {
	if !verbose {
		_, err := fn(query, args...)
		return err
	}

	ch := make(chan error)

	go func() {
		_, err := fn(query, args...)
		ch <- err
	}()

	t := time.Now()

	for {
		select {
		case err := <-ch:
			return err
		case <-time.Tick(time.Minute):
			log.Debug("Executing statement still in progress for %v", time.Since(t).Round(time.Second))
		}
	}
}

func clearStatement(s string) string {
	s = matchSQLComments.ReplaceAllString(s, ``)
	return matchEmptyEOL.ReplaceAllString(s, ``)
}
