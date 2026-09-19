package main

import (
	"html/template"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFiles("internal/web/templates/index.html"))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{"Title": "Hello World!!"}

		if err := tmpl.Execute(w, data); err != nil {
			log.Print(err)
		}
	})

	http.ListenAndServe(":8080", mux)
}
