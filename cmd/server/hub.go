package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// hub は接続中のブラウザをまとめて持ち、HTML の断片を全員に配る。
//
// 1 接続 = 1 チャネル。書き込みが詰まった接続は切り捨てる（遅い相手に全体を
// 引きずられないようにするため）。
type hub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}

	// snapshot は接続直後に送る HTML を返す。現在の一覧を丸ごと渡して
	// 「いま何が正しいか」を伝える。
	snapshot func() string
}

func newHub(snapshot func() string) *hub {
	return &hub{
		clients:  make(map[chan string]struct{}),
		snapshot: snapshot,
	}
}

func (h *hub) add() chan string {
	ch := make(chan string, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	n := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws: 接続 (%d 台)", n)
	return ch
}

func (h *hub) remove(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	n := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws: 切断 (%d 台)", n)
}

// broadcast は全接続に HTML を送る。
func (h *hub) broadcast(html string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- html:
		default: // 詰まっている接続は飛ばす
		}
	}
}

// serveWS は 1 接続を処理する。ブラウザからは何も受け取らず、送るだけ。
//
// 接続直後に最新の一覧を送る。htmx は画面が隠れている間 WebSocket を切るので
// （ws.pauseOnBackground の既定が true）、裏にいる間の変更は届かない。
// 再接続のたびに現在の状態を送り直すことで追いつかせる。通信断やスリープからの
// 復帰でも同じ経路で回復する。
func (h *hub) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Print("ws accept:", err)
		return
	}
	defer conn.CloseNow()

	ch := h.add()
	defer h.remove(ch)

	if h.snapshot != nil {
		ch <- h.snapshot()
	}

	// 受信は使わないが、閉じられたことを検知するために読み捨てる
	ctx := conn.CloseRead(r.Context())

	for {
		select {
		case <-ctx.Done():
			return
		case html := <-ch:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, []byte(html))
			cancel()
			if err != nil {
				return
			}
		}
	}
}
