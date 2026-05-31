package main

import (
	"gorm.io/gen"
	"gorm.io/gen/field"
)

//go:generate go run .

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:           "../../internal/dal/query",
		ModelPkgPath:      "../../internal/dal/model",
		Mode:              gen.WithQueryInterface,
		FieldSignable:     true,
		FieldNullable:     true,
		FieldCoverable:    false,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	wireGen, err := WireGen()
	if err != nil {
		panic(err)
	}
	defer wireGen.Close()
	g.UseDB(wireGen.DB)

	g.WithImportPkgPath(
		"github.com/shopspring/decimal",
		"gorm.io/datatypes",
		"gorm.io/plugin/soft_delete",
	)

	g.WithOpts(
		gen.FieldType("deleted_at", "soft_delete.DeletedAt"),
		gen.FieldGORMTag("deleted_at", func(tag field.GormTag) field.GormTag {
			tag.Set("softDelete", "milli")
			tag.Set("index", "")
			return tag
		}),
		gen.FieldJSONTag("deleted_at", "deleted_at,omitempty"),
	)

	commonOpts := []gen.ModelOpt{
		gen.FieldType("deleted_at", "soft_delete.DeletedAt"),
		gen.FieldGORMTag("deleted_at", func(tag field.GormTag) field.GormTag {
			tag.Set("softDelete", "milli")
			tag.Set("index", "")
			return tag
		}),
		gen.FieldJSONTag("deleted_at", "deleted_at,omitempty"),
	}

	g.ApplyBasic(
		g.GenerateModelAs("sys_cron_job", "CronJob", commonOpts...),
		g.GenerateModelAs("sys_cron_job_execution", "CronJobExecution", commonOpts...),
	)
	g.Execute()
}
