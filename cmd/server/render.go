package main

import (
	"bytes"
	"fmt"
	"io"
)

type Renderer struct {
	mode Mode
	tmpl *Tmpl
}

func (r *Renderer) Page(w io.Writer, name string, data map[string]any) error {
	templateData := map[string]any{
		"Env": r.mode,
	}

	for k, v := range data {
		templateData[k] = v
	}

	return r.Template(w, name, templateData)
}

func (r *Renderer) Template(w io.Writer, name string, data any) error {
	var buf bytes.Buffer
	if err := r.tmpl.Get().ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("render error: %w", err)
	}
	_, err := buf.WriteTo(w)
	return err
}
