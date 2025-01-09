package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yodigi7/pentago"
	"github.com/yodigi7/pentago-server/internal/server"
	"github.com/yodigi7/pentago-server/pkg/jsondefs"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// You can add checks here to allow only specific origins (like your frontend)
		return true
	},
}

var games sync.Map

func handleRead(conn *websocket.Conn, ch chan *jsondefs.Turn) {
	defer close(ch)
	for {
		var turn jsondefs.Turn
		if err := conn.ReadJSON(&turn); err != nil {
			log.Println("Error reading message:", err)
			break
		} else {
			ch <- &turn
		}
	}
	log.Println("Exiting handle read")
}

// will wait on the channel and the write to the connection whatever messages come in
func handleWrite(conn *websocket.Conn, gameChange chan []byte, writer chan []byte, done chan bool) {
	writerClosed := false
	for !writerClosed {
		select {
		case jsonMsg, ok := <-gameChange:
			if !ok {
				done <- true
				gameChange = nil
			} else {
				conn.WriteMessage(websocket.TextMessage, jsonMsg)
			}
		case jsonMsg, ok := <-writer:
			if !ok {
				writerClosed = true
				break
			} else {
				conn.WriteMessage(websocket.TextMessage, jsonMsg)
			}
		}
	}
	log.Println("Exiting handle write")
}

func writeRespToChan(ch chan []byte, errResp jsondefs.GeneralResponse) error {
	log.Printf("Sending message to channel: %s", errResp.Message)
	bytes, err := json.Marshal(errResp)
	// To prevent infinite recursion
	if err != nil && errResp.Code != 500 {
		log.Println("Error marshalling to bytes:", err)
		writeRespToChan(ch, jsondefs.GeneralResponse{
			Code:    500,
			Message: "Error marshalling to bytes",
		})
		return err
	}
	ch <- bytes
	return nil
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	log.Println("New client connected!")
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	var (
		color      pentago.Space
		gameChange chan []byte
		game       *server.Game
		payload    jsondefs.InitialConnection
	)
	if err = conn.ReadJSON(&payload); err != nil {
		log.Println("Error reading initial json payload")
		conn.WriteJSON(jsondefs.To400Response(err.Error()))
		return
	}

	color, gameChange, game, err = server.SetupNewClient(payload.GameId, &games)
	defer cleanupGame(game)
	if err != nil {
		log.Println("Error when setting up new client:", err)
		conn.WriteJSON(jsondefs.To400Response(err.Error()))
		return
	}
	conn.WriteJSON(jsondefs.InitialConnectionResponse{ColorNumber: color})

	bytes, err := json.Marshal(jsondefs.To200Response("All players connected"))
	if err != nil {
		log.Println("Error marshalling response when all players are connected:", err)
		return
	}
	if len(game.Channels) > 1 {
		for _, ch := range game.Channels {
			ch <- bytes
		}
	}

	writeChan := make(chan []byte)
	defer close(writeChan)
	readChan := make(chan *jsondefs.Turn)
	done := make(chan bool, 1)
	// Start go routing to send updates to client when the other player makes a move
	go handleWrite(conn, gameChange, writeChan, done)
	go handleRead(conn, readChan)

	// Start reading turns from the WebSocket
loop:
	for {
		select {
		case val, ok := <-done:
			{
				if !ok || val {
					break loop
				}
			}
		case turn, ok := <-readChan:
			{
				if !ok {
					break loop
				}
				if len(game.Channels) < 2 {
					err = writeRespToChan(writeChan, jsondefs.To400Response("Invalid move, not everyone is connected yet"))
					if err != nil {
						break loop
					}
					continue
				}
				if game.Turn != color {
					err = writeRespToChan(writeChan, jsondefs.To400Response("Invalid move, it's not your turn"))
					if err != nil {
						break loop
					}
					continue
				}
				err = game.PlaceMarble(turn.MarblePlacement.Row, turn.MarblePlacement.Col)
				if err != nil {
					err = writeRespToChan(writeChan, jsondefs.To400Response("Invalid move, unable to place marble there"))
					if err != nil {
						break loop
					}
					continue
				}
				err = game.RotateQuadrant(turn.Rotation.Quadrant, turn.Rotation.Direction)
				if err != nil {
					err = writeRespToChan(writeChan, jsondefs.To400Response("Invalid move, unable to rotate that way"))
					if err != nil {
						break loop
					}
					continue
				}
				game.UpdateChannels()
				if game.IsDraw {
					break loop
				}
				//TODO: remove when done debug
				log.Printf("Inner Game: %+v\n", *game.Game)
				log.Printf("Game: %+v\n", game)

			}
		}

	}

	if game != nil {
		if game.Winner == pentago.Empty && !game.IsDraw {
			writeRespToChan(writeChan, jsondefs.To200Response("YOU WIN"))
		}
	}
	conn.Close()
	log.Println("Exiting main handler")
}

func cleanupGame(g *server.Game) {
	g.CloseChannels()
	games.Delete(g.Id)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Serve the WebSocket at the /ws route
	http.HandleFunc("/ws", handleConnection)

	// Start the server
	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe failed:", err)
	}
}
