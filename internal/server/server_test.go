package server

import (
	// "reflect"
	"sync"
	"testing"

	"github.com/yodigi7/pentago"
)

func TestGameGetNextColor(t *testing.T) {
	t.Log("log test")
	game := newGame("game")
	space, err := game.getNextColor()
	if err != nil {
		t.Errorf("Expected error to be null but wasn't")
	}
	if space != pentago.White {
		t.Errorf("Expected next color to be white")
	}
	space, err = game.getNextColor()
	if err != nil {
		t.Errorf("Expected error to be null but wasn't")
	}
	if space != pentago.Black {
		t.Errorf("Expected next color to be black")
	}
	space, err = game.getNextColor()
	if err == nil {
		t.Errorf("Expected error but there was none")
	}
}

func TestUpdateChannels(t *testing.T) {
	t.Log("Finished setting up new client")
	gameId := "game"
	var games sync.Map

	var (
		gameChange chan []byte
		game       *Game
	)
	_, gameChange, game, _ = SetupNewClient(gameId, &games)
	_, gameChange, game, _ = SetupNewClient(gameId, &games)
	t.Log("Finished setting up new client")
	game.UpdateChannels()
	t.Log("Updated channels")
	bytes := <-gameChange
	t.Log("Read from channel")
	if bytes == nil {
		t.Errorf("Expected bytes to not be nil")
	}
}
