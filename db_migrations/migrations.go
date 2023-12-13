package db_migrations

type DatabaseSeedSpec struct {
	TableName string
	Fields    map[string]any
}

type DatabaseSeeds struct {
	Name         string
	Dependencies []string
	Seeds        []DatabaseSeedSpec
}
type DatabaseMigration struct {
	Name         string
	Dependencies []string
	MigrationDDL TableBlueprint
	AllowFailure bool
}
