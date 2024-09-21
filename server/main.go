package main

import (
	"encoding/json"
	"fmt"
	"gameserver/my_types"
	"math"
	"math/rand/v2"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/anthdm/hollywood/actor"
)

type PlayerSession struct {
	clientID  int
	username  string
	inLobby   bool // 在大厅内?
	conn      *websocket.Conn
	sessionID int
	ctx       *actor.Context
	serverPID *actor.PID
}

func newPlayerSession(serverPID *actor.PID, sessionID int, conn *websocket.Conn) actor.Producer {
	return func() actor.Receiver {
		return &PlayerSession{
			serverPID: serverPID,
			sessionID: sessionID,
			conn:      conn,
		}
	}
}

func (ps *PlayerSession) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		ps.ctx = c
		go ps.readloop()
	case *my_types.PlayerState:
		ps.sendPlayerState(msg)
	default:
		fmt.Println("unknown message type", msg)
	}
}

func (ps *PlayerSession) readloop() {
	var msg my_types.WsMessage
	for {
		if err := ps.conn.ReadJSON(&msg); err != nil {
			fmt.Println("read loop error: ", err)
			return
		}
		go ps.handleMessage(msg)
	}
}

func (ps *PlayerSession) handleMessage(msg my_types.WsMessage) {
	switch msg.Type {
	case "login":
		var loginMsg my_types.Login
		if err := json.Unmarshal(msg.Data, &loginMsg); err != nil {
			panic(err)
		}
		ps.clientID = loginMsg.ClientID
		ps.username = loginMsg.Username
	case "player_state":
		var pst my_types.PlayerState
		if err := json.Unmarshal(msg.Data, &pst); err != nil {
			panic(err)
		}
		pst.SessionID = ps.sessionID
		if ps.ctx != nil {
			ps.ctx.Send(ps.serverPID, &pst)
		}

	}
}

func (ps *PlayerSession) sendPlayerState(state *my_types.PlayerState) {
	bytes, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}

	msg := my_types.WsMessage{
		Type: "player_state",
		Data: bytes,
	}
	if err := ps.conn.WriteJSON(msg); err != nil {
		panic(err)
	}
}

type Server struct {
	ctx      *actor.Context
	sessions map[int]*actor.PID
}

func newServer() actor.Receiver {
	return &Server{
		sessions: make(map[int]*actor.PID),
	}
}

func (s *Server) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case *my_types.PlayerState:
		s.broadcast(c.Sender(), msg)
	case actor.Started:
		s.startHTTP()
		s.ctx = c
		_ = msg
	default:
		fmt.Println("unknown message type", msg)
	}
}

func (s *Server) broadcast(from *actor.PID, state *my_types.PlayerState) {
	for _, pid := range s.sessions {
		if !pid.Equals(from) {
			s.ctx.Send(pid, state)
		}
	}
}

func (s *Server) startHTTP() {
	fmt.Println("start http server on port 40000")
	go func() {
		http.HandleFunc("/ws", s.handleWs)
		http.ListenAndServe(":40000", nil)
	}()
}

// handle the upgrade
func (s *Server) handleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		// panic(err)
		fmt.Println("ws upgrade error: ", err)
		return
	}

	fmt.Println("a new client trying to connect")
	// fmt.Println("conn_info: ", conn)
	sid := rand.IntN(math.MaxInt)
	pid := s.ctx.SpawnChild(newPlayerSession(s.ctx.PID(), sid, conn), fmt.Sprintf("session_%d", sid))
	s.sessions[sid] = pid
}

func main() {
	e_config := actor.NewEngineConfig()

	e, err := actor.NewEngine(e_config)
	//e, err := actor.NewEngine()
	if err != nil {
		fmt.Println("create engine error: ", err)
		return
	}

	e.Spawn(newServer, "server")
	select {}
}
