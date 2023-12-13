package postgres

import (
	"github.com/convergence-platform/convergence-service-lib-for-go/db_migrations"
)

func PostgresSeedToSQL(blueprint []db_migrations.DatabaseSeedSpec) string {
	result := ""

	for _, seed := range blueprint {
		var fieldList []string
		var valuesList []string

		sep := ""

		for field, value := range seed.Fields {
			fieldList = append(fieldList, sep)
			valuesList = append(valuesList, sep)

			fieldList = append(fieldList, field)
			valuesList = append(valuesList, formatValue(value))
			sep = ", "
		}

		result += "INSERT INTO " + seed.TableName + "("
		for _, fieldToken := range fieldList {
			result += fieldToken
		}
		result += ") VALUES("
		for _, valueToken := range valuesList {
			result += valueToken
		}
		result += ");\n\n"
	}
	return result
}

func formatValue(o any) string {
	result := ""

	if o == nil {
		result = "NULL"
	} else if _, ok := o.(string); ok {
		result = o.(string)
	} else {
		panic("Unexpected type")
	}

	return result
}
