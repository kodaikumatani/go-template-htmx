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
| フロント | **htmx 4.0.0** のみ | Alpine.js / React は入れない。htmx の設計思想だけに集中する |
| CSS | 自前の `app.css` 1枚 | 見た目は最低限 |

### htmx のバージョンについて

**4.0.0（2026-08-28 リリース、v3 は欠番）を vendoring する。** 公式アナウンスに以下がある。

> we are not marking 4.0 as `latest` in NPM because we do not want to force-upgrade users who are
> relying on non-versioned CDN URLs. Instead, 2.x will remain `latest` and the 4.0 line will remain
> `next` until some point in early 2027. **The website, however, will reference 4.0.**

npm の `latest` が 2.x のままなのは移行期間の配慮であって、**ドキュメントサイトはすでに 4.0 基準**。
バージョン固定でファイルを vendoring するので `latest` タグは無関係。
公式 docs を引きながら書けるこの構成を採る。

ただし世の中の記事・書籍・サンプルコードはまだ 2.x なので、**コピペしたコードが動かないときは
まずバージョン差分を疑う**こと。2.x → 4.0 の差分で、このロードマップに効くのは以下。

| 項目 | htmx 2 | htmx 4 | 効く Step |
| --- | --- | --- | --- |
| 属性の継承 | 暗黙的に子孫へ継承 | **`:inherited` を付けないと継承しない**（`hx-target:inherited="this"`）。`:append` で追記も可 | 4, 5 |
| 4xx / 5xx レスポンス | swap しない | **swap する**。swap しないのは `204` と `304` のみ | 3, 8 |
| per-status の制御 | `htmx.config.responseHandling` | 属性 `hx-status:422="swap:innerHTML target:#errors"`（キーは `swap:` `target:` `select:` `push:` `replace:` `transition:`、`5xx` `50x` のワイルドカード可） | 3, 8 |
| イベント名 | `htmx:afterRequest` | `htmx:phase:action` 形式に統一 → `htmx:after:request`、`htmx:responseError` → `htmx:response:error` | 3, 8 |
| `hx-disabled-elt` | あり | **`hx-disable` に改名**（2.x の `hx-disable` は `hx-ignore` になった） | 5 |
| OOB の swap 順 | OOB が先 | **本体が先**、OOB と `<hx-partial>` が後（文書順） | 3 |
| `hx-delete` | 囲む form の値を送る | 送らない（`hx-get` と同じ）。必要なら `hx-include="closest form"` | 5 |
| history | localStorage にキャッシュ | キャッシュしない。戻ると再 fetch | 6 |
| timeout | 無制限 | 既定 60 秒（`htmx.config.defaultTimeout`） | — |
| `hx-ext` | 拡張の読み込みに必要 | 廃止。拡張は `<script>` を直接読むだけ | — |
| 内部実装 | `XMLHttpRequest` | `fetch()` | — |
| 新規 | — | `<hx-partial>`（1レスポンスで複数ターゲットを更新）、`hx-swap="innerMorph/outerMorph/textContent/delete"`、`HX-Request-Type: full\|partial` ヘッダ | 5, 7, 8 |

`hx-get` / `hx-post` / `hx-patch` / `hx-delete` / `hx-target` / `hx-swap`（`outerHTML` `afterbegin` `beforeend`）/
`hx-swap-oob` / `hx-trigger`（`changed` `delay:300ms`）/ `hx-include` / `hx-indicator` / `hx-confirm` /
`hx-push-url` / `hx-vals` / `HX-Request` ヘッダは 4.0 でも健在。

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
│           ├── htmx.min.js    # CDN ではなく vendoring（4.0.0, 36.7KB）
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
  - `static/htmx.min.js` は **htmx 4.0.0** を vendoring（jsdelivr/unpkg からダウンロード。
    `sha256:e484d917...`）。この時点では読み込んでいるだけで、属性は 1 つも使っていない
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
  - 検索フォーム。`hx-target` を 2 箇所に書かないよう、親に `:inherited` で持たせる（**v4 の書き方**）:
    ```html
    <div id="filters" hx-target:inherited="#issue-list" hx-indicator:inherited="#list-spinner">
      <input type="search" name="q"
             hx-get="/issues/list"
             hx-trigger="input changed delay:300ms, keyup[key=='Enter']"
             hx-include="#filters">
      <select name="status" hx-get="/issues/list" hx-trigger="change" hx-include="#filters">
        ...
      </select>
    </div>
    <span id="list-spinner" class="htmx-indicator">Searching...</span>
    ```
  - `input changed delay:300ms` が debounce 本体。`changed` があるので矢印キーなど値が変わらない打鍵は飛ばない
  - `hx-include="#filters"` で「q と status を常に両方送る」。どちらを触っても同じクエリが飛ぶ
- **学ぶこと**: debounce / fetch / state 更新が属性数行で終わる。JSON も useState も出てこない。
  併せて **v4 の明示的継承**を最初に体験しておく（2.x の記事をコピペすると `hx-target` が効かない理由がこれ）
- **完了条件**: 打つたびに一覧だけが差し替わる。ターミナルのログに `htmx=true path=/issues/list?...` が並ぶ。

## Step 3: 作成（POST → row を返す）

- **目的**: 「作ったものの HTML を返す」パターンとバリデーションエラー。
- やること
  - `POST /issues` → 成功時は `issues/row` を 1 件だけ返す:
    ```html
    <form hx-post="/issues"
          hx-target="#issue-list" hx-swap="afterbegin"
          hx-status:422="target:this swap:outerHTML"
          hx-on:htmx:after:request="this.reset()">
    ```
  - 送信後のフォームクリアは `hx-on:htmx:after:request="this.reset()"`
    （**v4 でイベント名が `htmx:after:request` に変わった**。2.x の `hx-on::after-request` は動かない）
  - バリデーションエラー時は **422 + フォーム fragment（エラーメッセージ入り）** を返す。
    v4 は 4xx/5xx も swap するので、素直に「エラー時はフォームを返す」で成立する。
    ただし成功時の target（`#issue-list` に追加）とは行き先が違うので、
    `hx-status:422="target:this swap:outerHTML"` で 422 のときだけフォーム自身を置き換える
  - Out of Band swap で件数バッジを同時更新:
    ```html
    <span id="issue-count" hx-swap-oob="true">{{ .Total }} issues</span>
    ```
    **v4 では本体 → OOB の順**に swap される（2.x は逆）。OOB が作った DOM を本体が前提にする書き方は避ける
- **学ぶこと**: 「1 レスポンスで複数箇所を更新する」の 2 つの流派。`hx-swap-oob`（返す HTML 側に印を付ける）と、
  v4 で追加された `<hx-partial hx-target="#count">`（レスポンスをターゲット単位で包む）。両方書いて比べる
- **完了条件**: リロードなしで先頭に行が増え、件数も同時に更新される。空タイトルで送ると
  フォームがエラー付きに差し替わる。

## Step 4: インライン編集（フォーム差し替え → 保存で row に戻す）

- **目的**: HTMX の設計思想が一番よく出るところ。
- やること
  - `GET /issues/{id}/edit` → row と同じ `id="issue-{{.ID}}"` を持つフォーム fragment を返す
  - 保存 `PATCH /issues/{id}` → `issues/row` を返す
  - キャンセルは `GET /issues/{id}` で表示状態の row を取り直す（クライアントに状態を持たせない）
  - row とフォームの両方に、**自分自身を置き換える**設定を `:inherited` で持たせる。公式の
    「Edit in Place」パターンがこの形（v4）:
    ```html
    <!-- partials/issues/row.html -->
    <div id="issue-{{ .ID }}" hx-target:inherited="this" hx-swap:inherited="outerHTML">
      <span>{{ .Title }}</span>
      <button hx-get="/issues/{{ .ID }}/edit">Edit</button>
    </div>

    <!-- partials/issues/form.html -->
    <form id="issue-{{ .ID }}" hx-patch="/issues/{{ .ID }}"
          hx-target:inherited="this" hx-swap:inherited="outerHTML">
      <input name="title" value="{{ .Title }}" autofocus>
      <button type="submit">Save</button>
      <button type="button" hx-get="/issues/{{ .ID }}">Cancel</button>
    </form>
    ```
    `hx-target="this"` を親要素に置き、中のボタンに継承させるのがコツ。ボタン 1 つ 1 つに
    target を書かなくて済む。**2.x では属性を書くだけで継承されたが、v4 は `:inherited` が必須**
- **学ぶこと**: 「UI の状態はサーバが返す HTML そのもの」。row / form が同じ id を持つのが鍵。
  編集中かどうかをブラウザ側のどこにも保存していない点を確認する。
- **完了条件**: Edit → 編集 → Save で行が戻る。Cancel でも戻る。

## Step 5: ステータス切り替えと削除

- やること
  - `PATCH /issues/{id}/status` → 更新後 row を返す（ボタンのラベルも Open/Close で入れ替わる）。
    Step 4 で row に `hx-target:inherited="this"` を付けてあるので、ボタンには
    `hx-patch="/issues/{{ .ID }}/status"` だけ書けばよい
  - `DELETE /issues/{id}` → 空ボディ + **200** を返して行を消す:
    ```html
    <button hx-delete="/issues/{{ .ID }}" hx-confirm="Delete?" hx-disable="this">Delete</button>
    ```
  - **落とし穴 1**: `204 No Content` は swap されない（v4 で swap されないのは `204` と `304` だけ）。
    空ボディでも 200 で返す。あるいは v4 の `hx-swap="delete"`（レスポンス内容を無視して要素を消す）を使う
  - **落とし穴 2**: 連打防止の属性は **v4 では `hx-disable`**。2.x の `hx-disabled-elt` から改名され、
    2.x の `hx-disable`（htmx の処理自体を無効化）は `hx-ignore` になった。ここは改名が交差しているので注意
  - **落とし穴 3**: v4 の `hx-delete` は囲む form の値を送らない。必要なら `hx-include="closest form"`
  - 余裕があれば `hx-swap="outerHTML swap:500ms"` + `.htmx-swapping` クラスの CSS transition で
    フェードアウトさせる（公式「Delete in Place」パターン）
- **完了条件**: Close 押下で当該行だけが置き換わる。削除で行が消える。

## Step 6: ページング + URL 同期

- やること
  - `ListResult{Items, Total, Page, PerPage, HasPrev, HasNext}` と `issues/pagination` template
  - ページリンクは `q` / `status` を保持したクエリを生成（template 関数 `queryString` を `FuncMap` に追加）
  - `hx-push-url="true"` で URL を書き換え、リロード・共有・戻るボタンに耐えるようにする
    → そのために `GET /issues` 側も `q` / `status` / `page` を読んで初期表示する必要がある（**重要**）
  - **v4 は履歴を localStorage にキャッシュしない**（2.x の悩みの種だった）。戻るボタンでは
    サーバに再リクエストが飛ぶので、「URL だけで状態が復元できる」実装が必須になる。
    逆に言えば、上の `GET /issues` をちゃんと作れば戻る/進むが自動で正しく動く
  - 余裕があれば「Load more」（`hx-swap="beforeend"` + 自分自身を置換）も実装して比較
- **完了条件**: 検索したまま 2 ページ目に行き、リロードしても戻るボタンでも同じ状態。

## Step 7: 詳細 drawer とコメント

- やること
  - `GET /issues/{id}/detail` → `hx-target="#drawer"` に詳細を差し込む
  - `POST /issues/{id}/comments` → `issues/comment` を 1 件返して `hx-swap="beforeend"`
  - drawer を閉じるボタンは JS なしなら `hx-get`（空を返す）か `<dialog>` + 少量の JS
  - コメント投稿で「コメント一覧の末尾に追加」と「一覧側の row のコメント数バッジ更新」を
    1 レスポンスで行う。v4 の `<hx-partial>` の出番:
    ```html
    <div>新しいコメント</div>
    <hx-partial hx-target="#issue-{{ .IssueID }}-comment-count" hx-swap="innerHTML">
      {{ .CommentCount }}
    </hx-partial>
    ```
- **学ぶこと**: fragment の入れ子（detail の中に comment list があり、comment 単体でも返せる）。
- **完了条件**: Detail → drawer 表示 → コメント投稿で末尾に増え、一覧側のバッジも更新される。

## Step 8: 仕上げ（エラー・UX・テスト）

- やること
  - エラーハンドリング: 500 のときトースト領域へ `HX-Retarget`（v4 でも仕様変更なし）+ `hx-swap-oob`。
    宣言的にやるなら `hx-status:5xx="target:#toast swap:innerHTML"`
  - `htmx:response:error` のグローバルハンドラ（**v4 で `htmx:responseError` から改名**）
  - `HX-Request` / `HX-Request-Type` ヘッダ判定方式へのリファクタを試す。
    v4 は `HX-Request-Type: full|partial` を自動で送るので、`/issues` と `/issues/list` を
    1 ハンドラに統合できる。明示ルート方式と比べてどちらが読みやすいか判断する
  - `httptest` でハンドラテスト。**HTML の中身をアサートする**（`strings.Contains` か
    `goquery` でセレクタ検証）。fragment が正しい id を返しているかがテストの主眼
  - `view.New` が全テンプレートを Parse できるかのテスト（`embed` した FS を渡すだけ）
  - live reload は `air` か `go run` の手動再起動で十分
- **完了条件**: `make check` が通る。

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
   htmx のターゲットは実質 API なので、ここが設計の中心になる。
6. **ネットの記事は 2.x 前提だと思って読む**。動かないときの容疑者トップ 3 は
   ①継承（`:inherited` が無い）②イベント名（`htmx:after:swap` 形式）③`hx-disabled-elt` → `hx-disable`。
   `htmx.config.implicitInheritance = true` で 2.x の継承挙動に戻せるが、学習目的なら戻さない方がいい。
