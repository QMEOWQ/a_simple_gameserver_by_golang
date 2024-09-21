package main

import (
	"encoding/json"
	"fmt"
	"gameserver/my_types"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

const wsServerEndpoint = "ws://localhost:40000/ws"

type Client struct {
	clientID int
	username string
	conn     *websocket.Conn
}

// func newClient(clientID int, username string) *Client {
// 	return &Client{
// 		clientID: clientID,
// 		username: username,
// 	}
// }

func newClient(username string, conn *websocket.Conn) *Client {
	return &Client{
		clientID: rand.Intn(math.MaxInt),
		username: username,
		conn:     conn,
	}
}

func (c *Client) login() error {
	bytes, err := json.Marshal(my_types.Login{
		ClientID: c.clientID,
		Username: c.username,
	})
	if err != nil {
		return err
	}

	msg := my_types.WsMessage{
		Type: "login",
		Data: bytes,
	}

	return c.conn.WriteJSON(msg)
}

func main() {
	// fmt.Println("whoo")
	// log.Println("whoo with a timestamp")
	// return

	dialer := websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, _, err := dialer.Dial(wsServerEndpoint, nil)
	if err != nil {
		log.Fatal(err)
	}

	c := newClient("Harden", conn)
	if err := c.login(); err != nil {
		log.Fatal(err)
	}

	go func() {
		var msg my_types.WsMessage
		for {
			if err := conn.ReadJSON(&msg); err != nil {
				fmt.Println("websocket read error: ", err)
				continue
			}

			switch msg.Type {
			case "player_state":
				var state my_types.PlayerState
				if err := conn.ReadJSON(&state); err != nil {
					fmt.Println("websocket read error: ", err)
					continue
				}
				fmt.Println("need to update player state: ", state)
			default:
				fmt.Println("received unknown message")
			}
		}
	}()

	for {
		x := rand.Intn(100)
		y := rand.Intn(100)
		state := my_types.PlayerState{
			Health:   100,
			Position: my_types.Position{X: x, Y: y},
		}

		bytes, err := json.Marshal(state)
		if err != nil {
			log.Fatal(err)
		}

		msg := my_types.WsMessage{
			Type: "player_state",
			Data: bytes,
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Microsecond * 100)
	}
}
