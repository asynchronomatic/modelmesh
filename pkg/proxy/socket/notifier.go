package socket

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64,
	WriteBufferSize: 64,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Notifier struct {
	lock    sync.Mutex
	clients map[*Client]bool
	action  chan *Action
}

func (n *Notifier) Broadcast() {
	select {
	case n.action <- &Action{Action: "broadcast"}:
	default:
	}
}

func (n *Notifier) Register(c *Client) {
	n.action <- &Action{Action: "register", Client: c}
}

func (n *Notifier) Deregister(c *Client) {
	n.action <- &Action{Action: "deregister", Client: c}
}

func (n *Notifier) Poll() {
	for action := range n.action {
		n.lock.Lock()
		switch action.Action {
		case "register":
			n.clients[action.Client] = true
		case "deregister":
			delete(n.clients, action.Client)
			action.Client.Close()
		case "broadcast":
			for client := range n.clients {
				client.Wakeup()
			}
		}
		n.lock.Unlock()
	}
}

func (n *Notifier) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		notifier: n,
		conn:     conn,
		wakeup:   make(chan bool, 8),
	}

	n.Register(client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func NewNotifier() *Notifier {
	return &Notifier{
		clients: make(map[*Client]bool),
		action:  make(chan *Action, 64),
	}
}
