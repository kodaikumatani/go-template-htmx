# go-template-htmx

Go の `html/template` と htmx だけでサーバサイドレンダリングの UI を組む練習用リポジトリ。

題材はミニ Issue Tracker。検索・部分更新・インライン編集・ページングを
ステップごとに積み上げていく。

## 構成

| 項目 | 採用 |
| --- | --- |
| ルーティング | `net/http`（Go 1.22+ の `ServeMux`。chi などは入れない） |
| テンプレート | `html/template` + `embed` |
| DB | SQLite（`modernc.org/sqlite`、cgo 不要） |
| フロント | htmx 4.0.0 を vendoring（Alpine.js / React は入れない） |

```
http.Handler → service → repository(SQLite) → domain → ViewModel → html/template → HTML → htmx
```

## 動かす

```sh
make run     # http://localhost:8080/issues
```

初回起動時に `issues.db` を作成し、サンプルの Issue を 7 件投入する。

| 環境変数 | 既定値 | 用途 |
| --- | --- | --- |
| `ADDR` | `:8080` | 待ち受けアドレス |
| `DB_DSN` | `file:issues.db?_pragma=journal_mode(WAL)&...` | SQLite の接続文字列 |

```sh
make check   # go vet + go test
make build   # bin/server に単一バイナリを出力
```

テンプレートと静的ファイルは `embed` でバイナリに埋め込んでいるので、
実行ディレクトリに依存しない。

## ディレクトリ

```
cmd/server/          エントリポイント
internal/domain/     中核の型（SQL も HTTP も知らない）
internal/service/    ユースケース。Repository interface はここで宣言する
internal/store/      SQLite 実装
internal/web/        ルーティング・ハンドラ
  ├─ view/           ViewModel と template 実行
  ├─ templates/      layout.html / pages/ / partials/
  └─ static/         htmx.min.js, app.css
docs/steps.md        ステップごとの計画と設計メモ
```

## 進め方

[docs/steps.md](docs/steps.md) に Step 0〜8 のロードマップがある。
1 ステップごとに「ブラウザで動く状態」にしてコミットする。
