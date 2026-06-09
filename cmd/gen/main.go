package main

import (
	"path/filepath"
	"runtime"

	"gorm.io/gen"
	"gorm.io/gen/field"
)

//go:generate go run .

func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func main() {
	root := projectRoot()
	g := gen.NewGenerator(gen.Config{
		OutPath:           filepath.Join(root, "internal/dal/query"),
		ModelPkgPath:      filepath.Join(root, "internal/dal/model"),
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

	roleModel := g.GenerateModelAs("sys_role", "Role")
	permModel := g.GenerateModelAs("sys_permission", "Permission")

	g.ApplyBasic(
		g.GenerateModelAs("sys_cron_job", "CronJob"),
		g.GenerateModelAs("sys_cron_job_execution", "CronJobExecution"),
		g.GenerateModelAs("sys_user", "User",
			gen.FieldJSONTag("password_hash", "-"),
			gen.FieldType("completed_tours", "datatypes.JSONSlice[string]"),
			gen.FieldRelate(field.Many2Many, "Roles", roleModel, &field.RelateConfig{
				GORMTag: field.GormTag{"many2many": {"sys_user_role"}},
			}),
		),
		g.GenerateModelAs("sys_user_role", "UserRole"),
		g.GenerateModelAs("sys_role", "Role",
			gen.FieldRelate(field.Many2Many, "Permissions", permModel, &field.RelateConfig{
				GORMTag: field.GormTag{"many2many": {"sys_role_permission"}},
			}),
			gen.FieldRelate(field.BelongsTo, "Parent", roleModel, &field.RelateConfig{
				RelatePointer: true,
				GORMTag:       field.GormTag{"foreignKey": {"ParentID"}},
			}),
			gen.FieldRelate(field.HasMany, "Children", roleModel, &field.RelateConfig{
				RelateSlice: true,
				GORMTag:     field.GormTag{"foreignKey": {"ParentID"}},
			}),
		),
		g.GenerateModelAs("sys_role_permission", "RolePermission"),
		g.GenerateModelAs("sys_permission", "Permission"),
	)
	g.Execute()
}
