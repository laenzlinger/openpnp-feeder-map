// SPDX-License-Identifier: GPL-3.0-or-later

package feedermap

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed template.html
var templateFS embed.FS

var funcMap = template.FuncMap{
	"lower": strings.ToLower,
}

func Render(w io.Writer, data *MapData) error {
	tmpl, err := template.New("template.html").Funcs(funcMap).ParseFS(templateFS, "template.html")
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	return tmpl.Execute(w, data)
}
