package internal

import (
	"bytes"
	"html/template"
	"net/http"
)

const (
	templateDir string = "web/tmpl"
)

type templateData struct {
	PageTitle string
	UID       string
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tmpl string, td *templateData) error {
	t, err := template.ParseFiles(
		templateDir+"/"+tmpl,
		templateDir+"/"+"base.html",
	)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	err = t.Execute(buf, td)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}
