package db_migrations

type BlueprintFormatter interface {
	ToSQL(blueprint any) string
}

type SqlDialectFormatter struct {
	CreateTableFormatter    BlueprintFormatter
	CreateRelationFormatter BlueprintFormatter
	InsertSeedsFormatter    BlueprintFormatter
}
