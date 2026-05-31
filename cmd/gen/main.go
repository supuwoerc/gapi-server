package main

//go:generate go run .

import (
	"gorm.io/gen"
	"gorm.io/gen/field"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:           "../../internal/dal/query",
		ModelPkgPath:      "../../internal/dal/model",
		Mode:              gen.WithQueryInterface,
		FieldSignable:     true,
		FieldNullable:     true,
		FieldCoverable:    false,
		FieldWithIndexTag: false,
		FieldWithTypeTag:  true,
	})
	wireGen, err := WireGen()
	if err != nil {
		panic(err)
	}
	defer wireGen.Close()
	g.UseDB(wireGen.DB)

	commonOpts := []gen.ModelOpt{
		gen.FieldType("deleted_at", "gorm.io/plugin/soft_delete.DeletedAt"),
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
