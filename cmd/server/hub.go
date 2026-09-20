package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// hub は接続中のブラウザに HTML の断片を配る。1 接続 = 1 チャネル。
type hub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}

	// snapshot は接続直後に送る HTML を返す。
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

// broadcast は全接続に HTML を送る。詰まっている接続は飛ばす。
func (h *hub) broadcast(html string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- html:
		default:
		}
	}
}

// serveWS は 1 接続を処理する。受信はせず、送るだけ。
//
// 接続直後に snapshot を送るのが重要。htmx はタブが隠れている間 WebSocket を切る
// （ws.pauseOnBackground の既定が true）ため、その間の変更は届かない。
// 再接続のたびに現在の状態を送り直して追いつかせる。通信断やスリープでも同じ。
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

	ctx := conn.CloseRead(r.Context()) // 受信は読み捨て、切断の検知だけに使う

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
