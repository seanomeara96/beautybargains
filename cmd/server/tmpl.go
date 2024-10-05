package main

import "html/template"

type Tmpl struct {
	tmpl *template.Template
	mode Mode
	glob string
}

func (t *Tmpl) Get() *template.Template {
	if t.mode == Prod && t.tmpl != nil {
		return t.tmpl
	}
	funcMap := template.FuncMap{
		"longDate":            formatLongDate,
		"placeholderImage":    placeholderImage,
		"truncateDescription": truncateDescription,
		"proper":              proper,
		"unescape":            unescape,
		"isCurrentPage":       isCurrentPage,
		"add":                 add,
		"subtract":            subtract,
		"lower":               lower,
	}
	t.tmpl = template.Must(template.New("web").Funcs(funcMap).ParseGlob(t.glob))
	return t.tmpl
}
