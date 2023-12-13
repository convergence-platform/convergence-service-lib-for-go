package postgres

import (
	"github.com/convergence-platform/convergence-service-lib-for-go/db_migrations"
	"strings"
)

func PostgresRelationshipToSQL(blueprint db_migrations.TablesRelationshipBlueprint) string {
	template := "ALTER TABLE {blueprint.table} ADD FOREIGN KEY ({blueprint.foreign_key}) REFERENCES {blueprint.primary_table}({blueprint.primary_key});"

	template = strings.Replace(template, "{blueprint.table}", blueprint.Table, -1)
	template = strings.Replace(template, "{blueprint.foreign_key}", blueprint.ForeignKey, -1)
	template = strings.Replace(template, "{blueprint.primary_table}", blueprint.PrimaryTable, -1)
	template = strings.Replace(template, "{blueprint.primary_key}", blueprint.PrimaryKey, -1)

	return template
}
