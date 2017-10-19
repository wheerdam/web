package main

import (
	"net/http"
	"html/template"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	t, _ := template.ParseFiles(dirTemplates + "/index.gtpl")
	t.Execute(w, nil)
}
