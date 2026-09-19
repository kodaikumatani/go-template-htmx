package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
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

// filterIssues は status と検索語 q で絞り込む。どちらも空なら全件。
func filterIssues(status, q string) []Issue {
	out := []Issue{}
	for _, i := range issues {
		if status != "" && i.Status != status {
			continue
		}
		// 大文字小文字を区別せずタイトルの部分一致を見る
		if q != "" && !strings.Contains(strings.ToLower(i.Title), strings.ToLower(q)) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func main() {
	mux := http.NewServeMux()

	// ファイルが 2 つになった。index.html の中から list.html を呼べる。
	tmpl := template.Must(template.ParseFiles(
		"internal/web/templates/index.html",
		"internal/web/templates/list.html",
	))

	// htmx.min.js を配る。/static/htmx.min.js で参照できる。
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("internal/web/static"))))

	// ページ全体を返す
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data := indexData{Title: "Issues", Issues: filterIssues("", "")}
		if err := tmpl.Execute(w, data); err != nil {
			log.Print(err)
		}
	})

	// 一覧の部分だけを返す。htmx が呼ぶのはこちら。
	// ExecuteTemplate で list.html だけを描くので、<html> や <head> は付かない。
	mux.HandleFunc("GET /issues/list", func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		q := r.URL.Query().Get("q")
		data := indexData{Title: "Issues", Issues: filterIssues(status, q)}

		log.Printf("fragment: q=%q status=%q -> %d 件", q, status, len(data.Issues))

		if err := tmpl.ExecuteTemplate(w, "list.html", data); err != nil {
			log.Print(err)
		}
	})

	log.Print("listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
