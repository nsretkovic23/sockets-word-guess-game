package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sockets-multiplayer/engine"
	"sockets-multiplayer/helpers"
)

// TODO: These should be in a config file, environment variables or command line arguments
const (
	SMILEY      = "\U0001F604"
	MAX_CONN    = 2
	MAX_LOBBIES = 1
	PORT        = 8080
	SECRET_WORD = "MY WORD"
)

type ServerMessage struct {
	Type            string
	Text            string
	Tag             int
	Turn            int
	PreviousGuesses []string
}

type ClientMessage struct {
	Sender int
	Guess  string
}

func main() {
	helpers.PrintInfo("Starting server...")

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", PORT))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer listener.Close()

	helpers.PrintInfo(fmt.Sprintf("Server started, listening on port %d", PORT))

	var lobbies []engine.Lobby
	for len(lobbies) < MAX_LOBBIES {

		lobby, err := engine.MakeLobby(listener, MAX_CONN, len(lobbies),
			engine.FormatMessage[*ServerMessage](&ServerMessage{
				"welcome",
				"Welcome to the game!",
				-1,
				-1,
				[]string{},
			}))
		if err != nil {
			fmt.Println("Error while trying to make a lobby", err)
			continue
		}

		fmt.Println("Lobby created", lobby.Id)

		lobbies = append(lobbies, *lobby)
		go runGame(lobby)
	}

	for len(lobbies) > 0 {

	}
}

type GameState struct {
	Turn            int
	GuessState      string
	PreviousGuesses []string
}

func runGame(lobby *engine.Lobby) {
	fmt.Println("Running game")
	if lobby.Conns == nil || len(lobby.Conns) <= 1 {
		helpers.PrintRed("Not enough players in the lobby")
		return
	}

	state := &GameState{0, "", []string{}}

	for i, conn := range lobby.Conns {
		msg := &ServerMessage{
			"tag_assignment",
			fmt.Sprintf("You are player %d", i),
			i,
			0,
			[]string{},
		}
		_, err := engine.SendUnicastMessage(&conn, engine.FormatMessage[*ServerMessage](msg))
		if err != nil {
			fmt.Println("Error sending message to player", err)
		}
	}

	for {
		msg := engine.FormatMessage(
			&ServerMessage{"turn", fmt.Sprintf("Player %d's turn", state.Turn),
				-1,
				state.Turn,
				state.PreviousGuesses,
			})
		engine.SendMulticastMessage(&lobby.Conns, msg)

		msgRaw := make([]byte, 2048)
		n, err := lobby.Conns[state.Turn].Read(msgRaw)
		if err != nil {
			// TODO: TEST THIS
			helpers.PrintRed("Client disconnected: " + err.Error())
			state.Turn = (state.Turn + 1) % len(lobby.Conns)
			break
		}

		var clientMsg ClientMessage

		err = json.Unmarshal(msgRaw[:n], &clientMsg)
		if err != nil {
			helpers.PrintRed("Error unmarshalling message: " + err.Error())
			continue
		}

		if clientMsg.Sender != state.Turn {
			helpers.PrintRed("Player " + fmt.Sprintf("%d", state.Turn) + " tried to guess out of turn")
			continue
		}

		if clientMsg.Guess == SECRET_WORD {
			msg := engine.FormatMessage(&ServerMessage{"game_over", "Game over!", -1, -1, state.PreviousGuesses})
			engine.SendMulticastMessage(&lobby.Conns, msg)
			break
		}

		state.PreviousGuesses = append(state.PreviousGuesses, clientMsg.Guess)
		helpers.PrintInfo("Player " + fmt.Sprintf("%d", state.Turn) + " guessed: " + clientMsg.Guess)
		state.Turn = (state.Turn + 1) % len(lobby.Conns)
	}

}