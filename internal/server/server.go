package server

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/yodigi7/pentago"
	"github.com/yodigi7/pentago-server/pkg/jsondefs"
)

type GameStateResponse struct {
	Game Game
}

type Game struct {
	*pentago.Game
	IsDraw    bool
	Id        string
	NextColor pentago.Space `json:"-"`
	Channels  []chan []byte `json:"-"`
	mut       sync.RWMutex  `json:"-"`
}

func (g *Game) UpdateChannels() error {
	g.mut.RLock()
	defer g.mut.RUnlock()
	for _, ch := range g.Channels {
		var bytes []byte
		bytes, err := json.Marshal(g)
		if err != nil {
			log.Println("Error marshalling to bytes:", err)

			bytes, innerErr := json.Marshal(jsondefs.GeneralResponse{
				Code:    500,
				Message: "Error marshalling to bytes",
			})
			if innerErr != nil {
				return innerErr
			}
			ch <- bytes
			return err
		}
		ch <- bytes
	}
	return nil
}

func (g *Game) CloseChannels() {
	g.mut.Lock()
	defer g.mut.Unlock()
	log.Println("Closing channels")
	for i, ch := range g.Channels {
		log.Printf("Closing channel: %d\n", i)
		close(ch)
	}
	g.Channels = make([]chan []byte, 0)
}

func newGame(id string) *Game {
	return &Game{
		Game:      pentago.NewGame(),
		Id:        id,
		NextColor: pentago.White,
		Channels:  make([]chan []byte, 0),
		IsDraw:    false,
	}
}

// Automatically check for winner and draw after successful rotate quadrant
func (g *Game) RotateQuadrant(q pentago.Quadrant, d pentago.RotationDirection) error {
	g.mut.Lock()
	defer g.mut.Unlock()
	err := g.Game.RotateQuadrant(q, d)
	if err != nil {
		return err
	}
	if w := g.Game.CheckForWinner(); w != pentago.Empty {
		g.Winner = w
	} else if g.Game.IsDraw() {
		g.IsDraw = true
	}
	return err
}

func (g *Game) addToGame() chan []byte {
	g.mut.Lock()
	defer g.mut.Unlock()

	newChannel := make(chan []byte, 1)
	g.Channels = append(g.Channels, newChannel)
	return newChannel
}

func (g *Game) getNextColor() (pentago.Space, error) {
	g.mut.Lock()
	defer g.mut.Unlock()

	switch g.NextColor {
	case pentago.White:
		g.NextColor = pentago.Black
		return pentago.White, nil
	case pentago.Black:
		g.NextColor = pentago.Empty
		return pentago.Black, nil
	default:
		return pentago.Black, errors.New("Already have max players connected")
	}
}

type PlayerConnection struct {
	GameId string
	Color  pentago.Space
}

// creates new game in games if not already created
// returns unique generated player UUID or error
func SetupNewClient(gameId string, games *sync.Map) (pentago.Space, chan []byte, *Game, error) {
	var color pentago.Space
	gameInterface, _ := games.LoadOrStore(gameId, newGame(gameId))
	game, ok := gameInterface.(*Game)
	if !ok {
		return pentago.Empty, nil, nil, errors.New("Error casting to game")
	}
	var err error
	color, err = game.getNextColor()
	if err != nil {
		return pentago.Empty, nil, nil, err
	}

	return color, game.addToGame(), game, nil
}
