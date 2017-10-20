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

// https://gist.github.com/d-schmidt/587ceec34ce1334a5e60
func redirectHTTP(w http.ResponseWriter, req *http.Request) {
    // remove/add not default ports from req.Host
    target := "https://" + req.Host + req.URL.Path 
    if len(req.URL.RawQuery) > 0 {
        target += "?" + req.URL.RawQuery
    }
    http.Redirect(w, req, target,
            // see @andreiavrammsd comment: often 307 > 301
            http.StatusTemporaryRedirect)
}