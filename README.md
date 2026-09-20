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
| エラー表示 | 済 | `hx-status:4xx:inherited` でエラー本文をバナーに集約 |
| リアルタイム同期 | 済 | `hx-ws:connect` + `<hx-partial>` を WebSocket で配信 |
| Open / Closed 切り替え | 未 | 行を返して置き換えるだけなので編集と同じ形になる |
| ページング | 未 | |
| コメント追加 | 未 | |

## リアルタイム同期

タブを 2 つ開いて片方で操作すると、もう片方が即座に変わる。
クライアントに書いた JavaScript は 0 行で、`<body hx-ws:connect="/ws">` の 1 属性だけ。

```
タブA で保存  → PATCH /issues/3      （HTTP。レスポンスは 204 で何も返さない）
                    ↓
              サーバがデータを更新し、更新後の行を全接続に配る
                    ↓ WebSocket
              <hx-partial hx-target="#issue-3" hx-swap="outerHTML"><li>…</li></hx-partial>
                    ↓
              タブA も タブB も同時に書き換わる
```

**変更は HTTP、画面の更新は WebSocket** と分担している。
変更系は 204（作成のみ OOB の空フォーム）を返し、行の HTML は必ず WS 経由で届く。
自分のタブも「その他大勢」の一員として扱うことで、更新の経路が 1 本になる。

WebSocket のメッセージには `hx-target` を指定するリクエスト側の要素が存在しないので、
**送る HTML 自身が行き先を持つ**（`<hx-partial>` は htmx 4 の新タグ）。

| 操作 | 送るもの |
| --- | --- |
| 作成 | `<hx-partial hx-target="#issue-list" hx-swap="afterbegin">` |
| 更新 | `<hx-partial hx-target="#issue-3" hx-swap="outerHTML">` |
| 削除 | `<hx-partial hx-target="#issue-3" hx-swap="delete">` |
| 接続時 | 一覧を丸ごと（下記） |

### 切れても追いつけるようにする

htmx は**タブが裏に回ると WebSocket を切る**（`ws.pauseOnBackground` の既定が `true`）。
そのため裏にいる間の変更は届かず、戻っても古いままになる。

対策として**接続時に現在の一覧を丸ごと送っている**。`ws.pauseOnBackground` を無効にする手もあるが、
それでは通信断・スリープ・サーバ再起動を防げない。「切れない前提」ではなく
「切れても追いつける」設計にしてある。

## 構成

```
cmd/server/main.go            ルーティングとハンドラ
cmd/server/hub.go             WebSocket の接続管理とブロードキャスト
internal/web/templates/
  index.html                  ページ全体
  list.html                   一覧（#issue-list ごと返す）
  row.html                    行 1 つ（単独でも返す）
  edit-row.html               編集フォーム（row と同じ id を持つ）
  new-form.html               作成フォーム（OOB でクリアする）
internal/web/static/
  htmx.min.js                 htmx 4.0.0 を vendoring
  hx-ws.min.js                WebSocket 拡張（htmx 4 では <script> を置くだけで有効）
  app.css
```

### エンドポイント

| メソッド・パス | 返すもの |
| --- | --- |
| `GET /` | ページ全体（クエリ `?q=&status=` を読んで復元する） |
| `GET /issues/list` | `<ul id="issue-list">` だけ |
| `POST /issues` | OOB の空フォーム（作った行は WebSocket で配る） |
| `GET /issues/{id}` | 表示状態の `<li>`（編集キャンセル用） |
| `GET /issues/{id}/edit` | 編集フォームの `<li>` |
| `PATCH /issues/{id}` | 204（更新後の行は WebSocket で配る） |
| `DELETE /issues/{id}` | 204（削除の指示は WebSocket で配る） |
| `GET /ws` | WebSocket。接続時に一覧、以降は変更のたびに `<hx-partial>` |

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
| 裏のタブに更新が届かない | htmx はタブが隠れると WebSocket を切る（`ws.pauseOnBackground`）。再接続時に状態を送り直して解決 |

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
