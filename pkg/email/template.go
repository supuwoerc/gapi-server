package email

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

type TemplateRenderer struct {
	templates *template.Template
}

func NewTemplateRenderer() *TemplateRenderer {
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	return &TemplateRenderer{templates: tmpl}
}

func (r *TemplateRenderer) Render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
