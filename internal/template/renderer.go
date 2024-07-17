package template

import (
	"bytes"
	"html/template"
	"net/http"
)

const (
	templateDir string = "web/tmpl"
)

type Data struct {
	PageTitle string
	UID       string
}

func Render(w http.ResponseWriter, r *http.Request, tmpl string, td any) error {
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
