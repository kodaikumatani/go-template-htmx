# go-template-htmx

**目的は htmx を体験すること。** Go の `html/template` が返す HTML だけで UI を動かし、
JavaScript を書かずにどこまでできるかを確かめる。

題材はミニ Issue Tracker。Todo より少しだけ複雑で、
「CRUD + 検索 + フィルタ + 部分更新」が一通り入るため htmx の主要パターンをほぼ全部踏める。

## 動かす

```sh
mise run dev     # ホットリロードで起動 → http://localhost:8080
```

| タスク | 内容 |
| --- | --- |
| `mise run dev` | air で起動。go / html / css を保存すると再ビルド |
| `mise run run` | 一度だけ起動 |
| `mise run build` | `bin/server` にビルド |
| `mise run check` | gofmt + go vet |

データは**メモリ上のスライス**。再起動すると初期状態に戻る。
DB を入れると htmx から関心が逸れるので、意図的に使っていない。

## 機能

| 機能 | 状態 | 使った htmx のパターン |
| --- | --- | --- |
| Issue 一覧 | 済 | — |
| 検索 | 済 | `hx-trigger="input changed delay:300ms"`（デバウンス） |
| ステータス絞り込み | 済 | `hx-include` で検索語と同時に送る |
| 新規作成 | 済 | 作った 1 行だけ返す + `hx-swap-oob` でフォームをクリア |
| タイトル編集 | 済 | 行 ⇄ 編集フォームを同じ id で差し替え |
| 削除 | 済 | 空ボディ + `hx-confirm` + フェードアウト |
| URL 同期 | 済 | レスポンスヘッダ `HX-Push-Url` |
| Open / Closed 切り替え | 未 | 行を返して置き換えるだけなので編集と同じ形になる |
| ページング | 未 | |
| コメント追加 | 未 | |

## 構成

```
cmd/server/main.go            ルーティングとハンドラ（全部ここ）
internal/web/templates/
  index.html                  ページ全体
  list.html                   一覧（#issue-list ごと返す）
  row.html                    行 1 つ（単独でも返す）
  edit-row.html               編集フォーム（row と同じ id を持つ）
  new-form.html               作成フォーム（OOB でクリアする）
internal/web/static/
  htmx.min.js                 htmx 4.0.0 を vendoring
  app.css
```

### エンドポイント

| メソッド・パス | 返すもの |
| --- | --- |
| `GET /` | ページ全体（クエリ `?q=&status=` を読んで復元する） |
| `GET /issues/list` | `<ul id="issue-list">` だけ |
| `POST /issues` | 作った `<li>` + OOB の空フォーム |
| `GET /issues/{id}` | 表示状態の `<li>`（編集キャンセル用） |
| `GET /issues/{id}/edit` | 編集フォームの `<li>` |
| `PATCH /issues/{id}` | 更新後の `<li>` |
| `DELETE /issues/{id}` | 空ボディ |

**返すのは常に HTML で、JSON は 1 度も出てこない。**

## 分かったこと

- **同じテンプレートを「ページの一部」と「単体のレスポンス」の両方に使う。**
  `list.html` も `row.html` も、初期表示と部分更新で共用している。ここを分けて書くとすぐ食い違う
- **画面の状態はサーバが返した HTML そのもの。** 「編集中かどうか」を持つ変数はどこにもない。
  編集のキャンセルすら `GET /issues/{id}` で元の行を取り直している
- **id が htmx の API になる。** `#issue-list` `#issue-{id}` `#new-issue` は
  サーバが返す HTML と `hx-target` の両方に現れる取り決め

### 踏んだ落とし穴

| 症状 | 原因 |
| --- | --- |
| リクエストが飛ばない | `hx-target` を `hx-garget` と誤記。htmx は不明な属性を黙って無視する |
| ボタンや一覧が消える | `hx-target` 未指定時の既定値は「自分自身」。`outerHTML` と組み合わせると要素ごと消える |
| 以降すべての操作が効かない | `outerHTML` で `id` を持つ要素ごと置き換えてしまい、`hx-target` の指す先が消えた |
| 削除しても行が残る | 空ボディを `204` で返した。htmx は 204 だけ swap しない。空でも `200` を返す |
| 画面にエラー文が貼られる | htmx 4 は 4xx/5xx も swap する。誤った URL への 405 の本文がそのまま表示された |
| 親に書いた `hx-get` が効かない | 動詞属性は継承されない。`:inherited` が効くのは `hx-target` などの修飾属性だけ |

## htmx のバージョン

**4.0.0**（2026-08-28 リリース、v3 は欠番）を vendoring している。
npm の `latest` は移行期間として 2.x のままだが、公式ドキュメントサイトはすでに 4.0 基準。

世の中の記事やサンプルは 2.x 前提なので、コピペしたコードが動かないときは以下を疑う。

- 属性の継承が明示的になった（`hx-target:inherited`）
- 4xx/5xx も swap される（swap しないのは 204 と 304 のみ）
- イベント名が `htmx:after:request` 形式に統一された
- `hx-disabled-elt` が `hx-disable` に改名（2.x の `hx-disable` は `hx-ignore` へ）

## 参考

`step-0-1` ブランチに、同じ題材を SQLite + domain/service/store の層構成で実装した版と、
`docs/steps.md` のロードマップがある（main とは別系統）。
