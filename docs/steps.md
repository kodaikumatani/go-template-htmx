# go template + HTMX 体験ロードマップ

## 題材

ミニ Issue Tracker。CRUD / 検索 / フィルタ / ページング / 部分更新 / ネストしたリソース（コメント）が
一通り入るので、HTMX の主要パターンをほぼ全部踏める。

## 技術選定

| 項目 | 採用 | 理由 |
| --- | --- | --- |
| ルーティング | `net/http` (標準) | Go 1.22+ の `ServeMux` が `"PATCH /issues/{id}"` 形式に対応済み。chi 不要 |
| テンプレート | `html/template` + `embed` | 本題。`embed.FS` で単一バイナリに |
| DB | SQLite (`modernc.org/sqlite`) | Pure Go で CGO 不要 |
| フロント | HTMX 2.x のみ | Alpine.js / React は入れない。HTMX の設計思想だけに集中する |
| CSS | 自前の `app.css` 1枚 | 見た目は最低限 |

## ディレクトリ構成（最終形）

```
.
├── go.mod
├── Makefile
├── cmd/server/main.go
├── internal/
│   ├── domain/            # Issue, Status, Comment（DB もHTTPも知らない）
│   │   └── issue.go
│   ├── service/           # ユースケース。Repository interface はここで定義（利用側定義）
│   │   └── issue.go
│   ├── store/             # SQLite 実装
│   │   ├── db.go
│   │   ├── schema.sql
│   │   └── issue.go
│   └── web/
│       ├── router.go           # //go:embed templates static はここ
│       ├── issue_handler.go
│       ├── view/              # ViewModel + template 実行
│       │   ├── render.go
│       │   └── model.go
│       ├── templates/
│       │   ├── layout.html         # define "layout"（{{template "content" .}} を持つ）
│       │   ├── pages/
│       │   │   └── issues/index.html   # define "content"
│       │   └── partials/
│       │       └── issues/             # define "issues/row" など一意な名前
│       │           ├── list.html
│       │           ├── row.html
│       │           ├── form.html
│       │           ├── detail.html
│       │           └── comment.html
│       └── static/
│           ├── htmx.min.js    # CDN ではなく vendoring（2.0.10）
│           └── app.css
```

> `//go:embed` は自パッケージ配下しか埋め込めないため、`templates/` と `static/` は
> リポジトリルートではなく `internal/web/` 配下に置いた。おかげで実行ディレクトリに
> 依存せず、単一バイナリで動く。
>
> `pages/` と `partials/` を分けているのは、ページごとに `"content"` という同じ
> テンプレート名を使い回すため。`view.New` が
> 「layout + 全 partials」のセットを `Clone()` して、そこにページを 1 枚だけ足す。
> fragment は `Partial(w, "issues/row", vm)` と名前指定で単独 Execute できる。

データの流れ:

```
http.Handler → service → repository(SQLite) → domain → ViewModel → html/template → HTML → HTMX
```

## ルーティング（最終形）

| メソッド・パス | 返すもの | HTMX 側の使い方 |
| --- | --- | --- |
| `GET /issues` | ページ全体 | 通常遷移 |
| `GET /issues/list` | `#issue-list` の中身だけ | 検索・フィルタ・ページング |
| `POST /issues` | 追加された 1 row | `hx-swap="afterbegin"` |
| `GET /issues/{id}` | row（表示状態） | 編集キャンセル用 |
| `GET /issues/{id}/edit` | row を置き換える編集フォーム | `hx-swap="outerHTML"` |
| `PATCH /issues/{id}` | 更新後の row | `hx-swap="outerHTML"` |
| `PATCH /issues/{id}/status` | 更新後の row | `hx-swap="outerHTML"` |
| `DELETE /issues/{id}` | 空ボディ（200） | `hx-swap="outerHTML"` で消す |
| `GET /issues/{id}/detail` | drawer 用の詳細 fragment | `hx-target="#drawer"` |
| `POST /issues/{id}/comments` | 追加されたコメント 1 件 | `hx-swap="beforeend"` |

---

# Steps

各 step の終わりに必ず「ブラウザで動く状態」になるようにしてある。
1 step ずつコミットする。

## Step 0: スケルトンと全ページレンダリング ✅ done

- **目的**: `html/template` + `embed` + layout 合成の型を作る。
- やったこと
  - `go mod init github.com/kodaikumatani/go-template-htmx`
  - `cmd/server/main.go`: タイムアウト設定済みの `http.Server` + SIGINT でのグレースフルシャットダウン。
    ポートは `ADDR`（既定 `:8080`）
  - `internal/web/router.go`: `http.ServeMux`（`GET /{$}` → `/issues` リダイレクト、`GET /issues`、
    `GET /static/`）＋ リクエストログ middleware（`HX-Request` の有無も出す）
  - `internal/web/view/render.go`: 起動時に全テンプレートを Parse。
    `Page(w, status, "issues/index", vm)` と `Partial(w, status, "issues/row", vm)` の 2 つの入口。
    レンダリングは一度 `bytes.Buffer` に書いてから出す（途中で失敗しても壊れた HTML を返さない）
  - `internal/web/view/model.go`: `IssuesIndex` / `Issue` の ViewModel
  - ダミーデータは `issue_handler.go` の `dummyIssues()`（Step 1 で置き換える）
  - `Makefile`（`run` / `build` / `test` / `vet` / `check`）、`.gitignore`
- **学んだこと**: layout 継承は「layout が `{{template "content" .}}` を呼び、
  ページ側が `{{define "content"}}` を持つ」形。ページごとに同名の `"content"` を使うので、
  1 つの巨大なテンプレートセットにはできず、**ベースセットを `Clone()` してページを 1 枚足す**構成になる。
- **完了条件**: `make run` → `http://localhost:8080/issues` にダミー一覧が出る（達成）。

## Step 1: 永続化層（SQLite）と 3 層構成

- **目的**: HTMX に入る前に「データの流れ」を通す。
- やること
  - `domain.Issue{ID, Title, Body, Status, CreatedAt, UpdatedAt}`、`domain.Status`（`open` / `closed`）
  - `store/schema.sql` + 起動時マイグレーション（`CREATE TABLE IF NOT EXISTS`）
  - `service.IssueRepository` interface を service 側に定義 → `store.IssueStore` が実装
  - `service.List(ctx, ListQuery) (ListResult, error)`：`ListQuery{Q, Status, Page, PerPage}`
  - seed データ投入用の小さなコマンドか、起動時 seed（件数 0 のときだけ）
  - `view/model.go` に ViewModel（`IssueVM{ID, Title, StatusLabel, IsOpen, CreatedAtHuman}`）。
    **domain をそのまま template に渡さない**のがポイント
- **完了条件**: 一覧が DB の中身で表示される。
- **注意**: SQLite は書き込み直列なので `db.SetMaxOpenConns(1)` か WAL 有効化（`_pragma=journal_mode(WAL)`）を入れておく。

## Step 2: fragment 化 + 検索 + ステータスフィルタ ★HTMX 初体験

- **目的**: 「ページ全体」と「一部分」を同じ template から返す、を体験する。
- やること
  - `GET /issues/list` を追加。`issues/list` template だけ Execute して返す
  - `issues/index.html` は `#issue-list` を持ち、初期表示は同じ `issues/list` を埋め込む（重複を作らない）
  - 検索 input:
    ```html
    <input name="q" hx-get="/issues/list"
           hx-trigger="keyup changed delay:300ms, search"
           hx-target="#issue-list" hx-include="#filters">
    ```
  - ステータス `<select name="status">` を `hx-trigger="change"` で同じ target へ
  - `hx-indicator` でローディング表示
- **学ぶこと**: debounce / fetch / state 更新が属性 3 行で終わる。JSON も useState も出てこない。
- **完了条件**: 打つたびに一覧だけが差し替わる。DevTools の Network に `text/html` が並ぶ。

## Step 3: 作成（POST → row を返す）

- **目的**: 「作ったものの HTML を返す」パターンとバリデーションエラー。
- やること
  - `POST /issues` → 成功時は `issues/row` を 1 件だけ返し、`hx-target="#issue-list" hx-swap="afterbegin"`
  - 送信後にフォームを空にする: `hx-on::after-request="if(event.detail.successful) this.reset()"`
  - バリデーションエラー時は **422 + フォーム fragment（エラーメッセージ入り）** を返す。
    HTMX はデフォルトで 4xx を swap しないので、`hx-target-error` 相当の処理か
    `htmx.config.responseHandling` の調整、あるいは 200 で返してフォームを置き換える方針を選ぶ
    → ここは「HTMX のエラー設計」を学ぶ山場なので、両方試してどちらが好みか決める
  - Out of Band swap で件数バッジを同時更新:
    ```html
    <span id="issue-count" hx-swap-oob="true">{{ .Total }} issues</span>
    ```
- **完了条件**: リロードなしで先頭に行が増え、件数も同時に更新される。

## Step 4: インライン編集（フォーム差し替え → 保存で row に戻す）

- **目的**: HTMX の設計思想が一番よく出るところ。
- やること
  - `GET /issues/{id}/edit` → row と同じ `id="issue-{{.ID}}"` を持つフォーム fragment を返す
  - 保存 `PATCH /issues/{id}` → `issues/row` を返す
  - キャンセルは `GET /issues/{id}` で表示状態の row を取り直す（クライアントに状態を持たせない）
  - 編集フォームは `hx-swap="outerHTML"`、`hx-target="#issue-{{.ID}}"` を **フォーム側に付ける**
- **学ぶこと**: 「UI の状態はサーバが返す HTML そのもの」。row / form が同じ id を持つのが鍵。
- **完了条件**: Edit → 編集 → Save で行が戻る。Cancel でも戻る。

## Step 5: ステータス切り替えと削除

- やること
  - `PATCH /issues/{id}/status` → 更新後 row を返す（ボタンのラベルも Open/Close で入れ替わる）
  - `DELETE /issues/{id}` → `hx-confirm="Delete?"`、`hx-target="closest .issue-row"`, `hx-swap="outerHTML"`
  - **落とし穴**: 204 No Content を返すと HTMX は swap しない。空ボディでも 200 で返す
  - 連打防止に `hx-disabled-elt="this"`
- **完了条件**: Close 押下で当該行だけが置き換わる。削除で行が消える。

## Step 6: ページング + URL 同期

- やること
  - `ListResult{Items, Total, Page, PerPage, HasPrev, HasNext}` と `issues/pagination` template
  - ページリンクは `q` / `status` を保持したクエリを生成（template 関数 `queryString` を `FuncMap` に追加）
  - `hx-push-url="true"` で URL を書き換え、リロード・共有・戻るボタンに耐えるようにする
    → そのために `GET /issues` 側も `q` / `status` / `page` を読んで初期表示する必要がある（重要）
  - 余裕があれば「Load more」（`hx-swap="beforeend"` + 自分自身を置換）も実装して比較
- **完了条件**: 検索したまま 2 ページ目に行き、リロードしても同じ状態。

## Step 7: 詳細 drawer とコメント

- やること
  - `GET /issues/{id}/detail` → `hx-target="#drawer"` に詳細を差し込む
  - `POST /issues/{id}/comments` → `issues/comment` を 1 件返して `hx-swap="beforeend"`
  - drawer を閉じるボタンは JS なしなら `hx-get`（空を返す）か `<dialog>` + 少量の JS
- **学ぶこと**: fragment の入れ子（detail の中に comment list があり、comment 単体でも返せる）。
- **完了条件**: Detail → drawer 表示 → コメント投稿で末尾に増える。

## Step 8: 仕上げ（エラー・UX・テスト）

- やること
  - エラーハンドリング: 500 / 404 のときトースト領域へ `HX-Retarget` + `hx-swap-oob` でフラッシュ表示
  - `htmx:responseError` のグローバルハンドラ
  - `httptest` でハンドラテスト。**HTML の中身をアサートする**（`strings.Contains` か
    `goquery` でセレクタ検証）。fragment が正しい id を返しているかがテストの主眼
  - template が全部 Parse できるかの起動時チェック（`template.Must`）＋テスト
  - `Makefile`（`run` / `test` / `lint`）、live reload は `air` か `go run` の手動再起動で十分
- **完了条件**: `make test` が通る。

---

## 意識しておくポイント

1. **fragment は route で明示的に分ける**（`/issues/list`）。最初から `HX-Request` ヘッダで
   分岐させると「今どっちを返してるのか」が分かりにくい。Step 8 で ヘッダ判定方式に
   リファクタして比較するのが理解に効く。
2. **同じ HTML を 2 箇所に書かない**。初期表示も部分更新も同じ `issues/list` / `issues/row` を使う。
   ここを守れないと HTMX の旨みが消える。
3. **クライアントに状態を持たせない**。編集中かどうかも「サーバが返した HTML」で表現する。
4. **ViewModel を挟む**。`html/template` はロジックが書きづらいので、
   整形は Go 側（`StatusLabel`, `CreatedAtHuman`, `IsOpen`）で済ませる。
5. **id の命名規則を決める**: `#issue-list`, `#issue-{id}`, `#drawer`, `#toast`。
   HTMX のターゲットは実質 API なので、ここが設計の中心になる。
