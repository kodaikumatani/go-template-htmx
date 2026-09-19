package main

import (
	"html/template"
	"log"
	"net/http"
)

// Issue は課題 1 件。DB はまだ使わず、メモリ上のスライスに持つ。
type Issue struct {
	ID     int
	Title  string
	Status string // "open" または "closed"
}

var issues = []Issue{
	{ID: 1, Title: "検索が遅い", Status: "open"},
	{ID: 2, Title: "ログイン後にリダイレクトされない", Status: "open"},
	{ID: 3, Title: "READMEのセットアップ手順を更新", Status: "closed"},
}

// indexData はテンプレートに渡すデータ。テンプレートの中の . がこれになる。
type indexData struct {
	Title  string
	Issues []Issue
}

func main() {
	mux := http.NewServeMux()

	tmpl := template.Must(template.ParseFiles("internal/web/templates/index.html"))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := indexData{
			Title:  "Issues",
			Issues: issues,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Print(err)
		}
	})

	log.Print("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
