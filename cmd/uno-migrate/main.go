package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime/debug"
	"text/template"

	"github.com/glugox/unogo/db"
	"github.com/glugox/unogo/db/migration"
)

const (
	defaultMigrationDir  = "./migrations"
	envGooseDriver       = "MIGRATION_DRIVER"
	envGooseDBString     = "MIGRATION_DBSTRING"
	envGooseMigrationDir = "MIGRATION_DIR"
)

var (
	flags        = flag.NewFlagSet("uno-migrate", flag.ExitOnError)
	dir          = flags.String("dir", defaultMigrationDir, "directory with migration files")
	table        = flags.String("table", "migrations", "migrations table name")
	verbose      = flags.Bool("v", false, "enable verbose mode")
	help         = flags.Bool("h", false, "print help")
	version      = flags.Bool("version", false, "print version")
	certfile     = flags.String("certfile", "", "file path to root CA's certificates in pem format (only support on mysql)")
	sequential   = flags.Bool("s", false, "use sequential numbering for new migrations")
	allowMissing = flags.Bool("allow-missing", false, "applies missing (out-of-order) migrations")
	sslcert      = flags.String("ssl-cert", "", "file path to SSL certificates in pem format (only support on mysql)")
	sslkey       = flags.String("ssl-key", "", "file path to SSL key in pem format (only support on mysql)")
	noVersioning = flags.Bool("no-versioning", false, "apply migration commands with no versioning, in file order, from directory pointed to")
)
var (
	migrationVersion = ""
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	if *version {
		if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo != nil && migrationVersion == "" {
			migrationVersion = buildInfo.Main.Version
		}
		fmt.Printf("migration version:%s\n", migrationVersion)
		return
	}
	if *verbose {
		migration.SetVerbose(true)
	}
	if *sequential {
		migration.SetSequential(true)
	}
	migration.SetTableName(*table)

	args := flags.Args()
	if len(args) == 0 || *help {
		flags.Usage()
		return
	}
	// The -dir option has not been set, check whether the env variable is set
	// before defaulting to ".".
	if *dir == defaultMigrationDir && os.Getenv(envGooseMigrationDir) != "" {
		*dir = os.Getenv(envGooseMigrationDir)
	}

	switch args[0] {
	case "init":
		if err := migrationInit(*dir); err != nil {
			log.Fatalf("migration run: %v", err)
		}
		return
	case "create":
		if err := migration.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("migration run: %v", err)
		}
		return
	case "fix":
		if err := migration.Run("fix", nil, *dir); err != nil {
			log.Fatalf("migration run: %v", err)
		}
		return
	}

	args = mergeArgs(args)
	if len(args) < 3 {
		flags.Usage()
		return
	}

	driver, dbstring, command := args[0], args[1], args[2]
	// To avoid breaking existing consumers, treat sqlite3 as sqlite.
	// An implementation detail that consumers should not care which
	// underlying driver is used. Internally uses the CGo-free port
	// of SQLite: modernc.org/sqlite
	if driver == "sqlite3" {
		driver = "sqlite"
	}
	db, err := migration.OpenDBWithDriver(driver, db.NormalizeDBString(driver, dbstring, *certfile, *sslcert, *sslkey))
	if err != nil {
		log.Fatalf("-dbstring=%q: %v\n", dbstring, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("migration: failed to close DB: %v\n", err)
		}
	}()

	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	options := []migration.OptionsFunc{}
	if *allowMissing {
		options = append(options, migration.WithAllowMissing())
	}
	if *noVersioning {
		options = append(options, migration.WithNoVersioning())
	}
	if err := migration.RunWithOptions(
		command,
		db,
		*dir,
		arguments,
		options...,
	); err != nil {
		log.Fatalf("migration run: %v", err)
	}
}

func mergeArgs(args []string) []string {
	if len(args) < 1 {
		return args
	}
	if d := os.Getenv(envGooseDriver); d != "" {
		args = append([]string{d}, args...)
	}
	if d := os.Getenv(envGooseDBString); d != "" {
		args = append([]string{args[0], d}, args[1:]...)
	}
	return args
}

func usage() {
	fmt.Println(usagePrefix)
	flags.PrintDefaults()
	fmt.Println(usageCommands)
}

var (
	usagePrefix = `Usage: uno-migration [OPTIONS] DRIVER DBSTRING COMMAND
or
Set environment key
MIGRATION_DRIVER=DRIVER
MIGRATION_DBSTRING=DBSTRING
Usage: uno-migration [OPTIONS] COMMAND
Drivers:
    postgres
    mysql
    sqlite3
    mssql
    redshift
    tidb
    clickhouse
Examples:
    uno-migration sqlite3 ./foo.db status
    uno-migration sqlite3 ./foo.db create init sql
    uno-migration sqlite3 ./foo.db create add_some_column sql
    uno-migration sqlite3 ./foo.db create fetch_user_data go
    uno-migration sqlite3 ./foo.db up
    uno-migration postgres "user=postgres dbname=postgres sslmode=disable" status
    uno-migration mysql "user:password@/dbname?parseTime=true" status
    uno-migration redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status
    uno-migration tidb "user:password@/dbname?parseTime=true" status
    uno-migration mssql "sqlserver://user:password@dbname:1433?database=master" status
    uno-migration clickhouse "tcp://127.0.0.1:9000" status
    MIGRATION_DRIVER=sqlite3 MIGRATION_DBSTRING=./foo.db uno-migration status
    MIGRATION_DRIVER=sqlite3 MIGRATION_DBSTRING=./foo.db uno-migration create init sql
    MIGRATION_DRIVER=postgres MIGRATION_DBSTRING="user=postgres dbname=postgres sslmode=disable" uno-migration status
    MIGRATION_DRIVER=mysql MIGRATION_DBSTRING="user:password@/dbname" uno-migration status
    MIGRATION_DRIVER=redshift MIGRATION_DBSTRING="postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" uno-migration status
Options:
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-by-one            Migrate the DB up by 1
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
`
)

var sqlMigrationTemplate = template.Must(template.New("sql_migration").Parse(`-- Thank you for giving migration a try!
-- 
-- This file was automatically created running migration init. If you're familiar with migration
-- feel free to remove/rename this file, write some SQL and migration up. Briefly,
-- 
--
-- A single migration .sql file holds both Up and Down migrations.
-- 
-- All migration .sql files are expected to have a -- +migration Up directive.
-- The -- +migration Down directive is optional, but recommended, and must come after the Up directive.
-- 
-- The -- +migration NO TRANSACTION directive may be added to the top of the file to run statements 
-- outside a transaction. Both Up and Down migrations within this file will be run without a transaction.
-- 
-- More complex statements that have semicolons within them must be annotated with 
-- the -- +migration StatementBegin and -- +migration StatementEnd directives to be properly recognized.
-- 
-- Use GitHub issues for reporting bugs and requesting features, enjoy!
-- +migration Up
SELECT 'up SQL query';
-- +migration Down
SELECT 'down SQL query';
`))

// initDir will create a directory with an empty SQL migration file.
func migrationInit(dir string) error {
	if dir == "" || dir == defaultMigrationDir {
		dir = "migrations"
	}
	_, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err == nil, errors.Is(err, fs.ErrExist):
		return fmt.Errorf("directory already exists: %s", dir)
	default:
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return migration.CreateWithTemplate(nil, dir, sqlMigrationTemplate, "initial", "sql")
}
