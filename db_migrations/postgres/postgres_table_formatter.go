package postgres

import (
	"github.com/convergence-platform/convergence-service-lib-for-go/db_migrations"
	"strings"
)

func PostgresTableToSQL(blueprint db_migrations.TableBlueprint) string {
	result := ""

	checkExistence := ""
	if blueprint.CheckExistence {
		checkExistence = "IF NOT EXISTS "
	}

	result += "CREATE TABLE "
	result += checkExistence
	result += blueprint.Name
	result += " \n("

	constraintCount := 0
	for _, column := range blueprint.Columns {
		if column.IsUnique {
			constraintCount += 1
		}
	}

	index := 0
	for _, column := range blueprint.Columns {
		index += 1
		isLastStatement := index == len(blueprint.Columns) && constraintCount == 0
		result += "\n    "
		result += getColumnDefinition(column)
		if !isLastStatement {
			result += ","
		}
	}

	index = 0
	for _, column := range blueprint.Columns {
		if column.IsUnique {
			if index == 0 {
				result += "\n"
			}
			index += 1
			isLastStatement := index == constraintCount
			sql := "\n    CONSTRAINT {blueprint.name}_{column.name}_unique_index UNIQUE ({column.name})"
			sql = strings.Replace(sql, "{blueprint.name}", blueprint.Name, -1)
			sql = strings.Replace(sql, "{column.name}", column.Name, -1)
			result += sql

			if !isLastStatement {
				result += ","
			}
		}
	}

	result += "\n)"

	if len(blueprint.Indices) > 0 {
		for _, indexBlueprint := range blueprint.Indices {
			result += "\n\n"
			result += formatTableIndex(blueprint, indexBlueprint)
		}
	}

	return result
}

func formatTableIndex(blueprint db_migrations.TableBlueprint, indexBlueprint db_migrations.TableIndexBlueprint) string {
	table := blueprint.Name
	fieldsComma := ""
	fieldsUnderscore := ""

	sc := ""
	su := ""

	for _, col := range indexBlueprint.Columns {
		fieldsComma += sc
		fieldsComma += col
		fieldsUnderscore += su
		fieldsUnderscore += col
		sc = ", "
		su = "_"
	}

	result := "CREATE INDEX {table}_{fields_underscore}_index ON {table} USING {index_blueprint.type}({fields_comma})"

	result = strings.Replace(result, "{table}", table, -1)
	result = strings.Replace(result, "{fields_underscore}", fieldsUnderscore, -1)
	result = strings.Replace(result, "{index_blueprint.type}", indexBlueprint.Type, -1)
	result = strings.Replace(result, "{fields_comma}", fieldsComma, -1)

	return result
}

func getColumnDefinition(column db_migrations.TableColumnBlueprint) string {
	result := column.Name + " " + toSqlType(column.Type) + " "

	if column.IsPrimaryKey {
		result += "PRIMARY KEY "
	}

	if column.AllowNull {
		result += "NULL"
	} else {
		result += "NOT NULL"
	}

	if column.DefaultValue != "" {
		result += " DEFAULT "
		result += column.DefaultValue
	}

	return result
}

func toSqlType(columnType string) string {
	result := ""

	if columnType == "String" {
		result = "text"
	} else if strings.HasPrefix(columnType, "String[") && strings.HasSuffix(columnType, "]") {
		length := columnType[7 : len(columnType)-1]
		result = "varchar(" + length + ")"
	} else if columnType == "bool" || columnType == "boolean" {
		result = "boolean"
	} else if strings.ToLower(columnType) == "json" {
		result = "json"
	} else if columnType == "int" {
		result = "int"
	} else if columnType == "date" {
		result = "date"
	} else if columnType == "double" {
		result = "float8"
	} else if columnType == "float" {
		result = "float4"
	} else if columnType == "timestamp" {
		result = "timestamp"
	} else if strings.ToLower(columnType) == "uuid" {
		result = "uuid"
	}

	return result
}
