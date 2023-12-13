package db_migrations

type TableColumnBlueprint struct {
	Name         string
	Type         string
	IsPrimaryKey bool
	IsUnique     bool
	AllowNull    bool
	DefaultValue string
}

type TableIndexBlueprint struct {
	Type    string
	Columns []string
}

type TableBlueprint struct {
	Name           string
	CheckExistence bool
	Columns        []TableColumnBlueprint
	Indices        []TableIndexBlueprint
}

type TablesRelationshipBlueprint struct {
	Table        string
	ForeignKey   string
	PrimaryTable string
	PrimaryKey   string
}

func NewTableColumnBlueprint(name string, columnType string) *TableColumnBlueprint {
	result := new(TableColumnBlueprint)

	result.Name = name
	result.Type = columnType
	result.IsPrimaryKey = false
	result.IsUnique = false
	result.AllowNull = false
	result.DefaultValue = ""

	return result
}

func NewTableColumnBlueprintDetailed(name string, columnType string, isPrimaryKey bool, isUnique bool, allowNull bool, defaultValue string) *TableColumnBlueprint {
	result := new(TableColumnBlueprint)

	result.Name = name
	result.Type = columnType
	result.IsPrimaryKey = isPrimaryKey
	result.IsUnique = isUnique
	result.AllowNull = allowNull
	result.DefaultValue = defaultValue

	return result
}

func (table TableBlueprint) AddOperationDates(createdAt bool, updatedAt bool, deletedAt bool) {
	if createdAt {
		column := NewTableColumnBlueprintDetailed("created_at",
			"timestamp",
			false,
			false,
			false,
			"CURRENT_TIMESTAMP")
		table.Columns = append(table.Columns, *column)
	}
	if updatedAt {
		column := NewTableColumnBlueprintDetailed("updated_at",
			"timestamp",
			false,
			false,
			true,
			"")
		table.Columns = append(table.Columns, *column)
	}
	if deletedAt {
		column := NewTableColumnBlueprintDetailed("deleted_at",
			"timestamp",
			false,
			false,
			true,
			"")
		table.Columns = append(table.Columns, *column)
	}
}
