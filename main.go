package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
	"github.com/spf13/viper"
)

// Config struct to hold configurable parameters
type Config struct {
	EnginePath string
	ServerAddr string
	MoveTime   time.Duration
}

var (
	uciEngine *uci.Engine
	game      *chess.Game
	config    Config
)

func init() {
	// Initialize viper for configuration
	// viper.SetDefault("engine_path", "stockfish")
	viper.SetDefault("server_addr", ":8080")
	viper.SetDefault("move_time", 10)

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // or viper.SetConfigType("YAML")
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, using defaults: %v", err)
	}

	// Load configuration into the config struct
	config = Config{
		EnginePath: viper.GetString("engine_path"),
		ServerAddr: viper.GetString("server_addr"),
		MoveTime:   time.Duration(viper.GetInt("move_time")) * time.Millisecond,
	}

	// Start the UCI engine in a background process
	engine, err := uci.New(config.EnginePath)
	if err != nil {
		log.Fatalf("Failed to create UCI engine: %v", err)
	}

	// Set UCI options as per config
	options := viper.GetStringMap("uci_options")
	for key, value := range options {
		if err := engine.Run(uci.CmdSetOption{Name: key, Value: fmt.Sprintf("%v", value)}); err != nil {
			log.Fatalf("Failed to set UCI option %s: %v", key, err)
		}
	}

	if err := engine.Run(uci.CmdUCI, uci.CmdIsReady, uci.CmdUCINewGame); err != nil {
		log.Fatalf("Failed to run UCI engine: %v", err)
	}

	uciEngine = engine
	game = chess.NewGame() // Initialize the game globally
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	uciMove := r.URL.Query().Get("uci")
	sanMove := r.URL.Query().Get("san")

	var userMove *chess.Move
	var err error

	if uciMove != "" {
		// Try to decode the move as UCI
		userMove, err = chess.UCINotation{}.Decode(game.Position(), uciMove)
		if err != nil {
			http.Error(w, "Invalid UCI move", http.StatusBadRequest)
			return
		}
	} else if sanMove != "" {
		// Try to decode the move as SAN
		userMove, err = chess.AlgebraicNotation{}.Decode(game.Position(), sanMove)
		if err != nil {
			http.Error(w, "Invalid SAN move", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Move not specified", http.StatusBadRequest)
		return
	}

	if err := game.Move(userMove); err != nil {
		http.Error(w, "Failed to apply user's move", http.StatusInternalServerError)
		return
	}

	// Set the position after user's move
	cmdPos := uci.CmdPosition{Position: game.Position()}
	if err := uciEngine.Run(cmdPos); err != nil {
		http.Error(w, "Failed to set position", http.StatusInternalServerError)
		return
	}

	// Start the engine to calculate the best move
	cmdGo := uci.CmdGo{MoveTime: config.MoveTime}
	if err := uciEngine.Run(cmdGo); err != nil {
		http.Error(w, "Failed to start engine search", http.StatusInternalServerError)
		return
	}

	// Get the best move
	bestMove := uciEngine.SearchResults().BestMove
	if err := game.Move(bestMove); err != nil {
		http.Error(w, "Failed to apply best move", http.StatusInternalServerError)
		return
	}

	// Respond with the best move as JSON
	w.Header().Set("Content-Type", "application/json")
	legalMoves := game.Position().ValidMoves()
	legalMovesStr := make([]string, len(legalMoves))
	for i, move := range legalMoves {
		legalMovesStr[i] = move.String()
	}

	// Get the current game moves in SAN format
	moveHistory := game.MoveHistory()
	sanMoves := make([]string, len(moveHistory))
	for i, move := range moveHistory {
		sanMoves[i] = chess.AlgebraicNotation{}.Encode(move.PrePosition, move.Move)
	}

	response := map[string]interface{}{
		"best_move":   bestMove.String(),
		"fen":         game.Position().String(),
		"legal_moves": legalMovesStr,
		"info":        uciEngine.SearchResults().Info,
		"pgn":         sanMoves,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}
}

func handleShow(w http.ResponseWriter, r *http.Request) {
	// Get the current position in FEN format
	ascii := game.Position().Board().Draw()

	// Respond with the current board position in ascii
	fmt.Fprintf(w, "Current position: %s", ascii)
}

func handleFen(w http.ResponseWriter, r *http.Request) {
	// Get the current position in FEN format
	fen := game.Position().String()

	// Get the legal moves
	legalMoves := game.Position().ValidMoves()
	legalMovesStr := make([]string, len(legalMoves))
	for i, move := range legalMoves {
		legalMovesStr[i] = move.String()
	}

	// Get the current game moves in SAN format
	moveHistory := game.MoveHistory()
	sanMoves := make([]string, len(moveHistory))
	for i, move := range moveHistory {
		sanMoves[i] = chess.AlgebraicNotation{}.Encode(move.PrePosition, move.Move)
	}

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")

	// Create a response map including legal moves and SAN moves
	response := map[string]interface{}{
		"fen":         fen,
		"legal_moves": legalMovesStr,
		"pgn":         sanMoves,
	}

	// Encode the response as JSON and write it to the response writer
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}
}

func handleNewGame(w http.ResponseWriter, r *http.Request) {
	// Initialize a new game
	game = chess.NewGame()

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")

	// Create a response map
	response := map[string]string{"message": "New game started"}

	// Encode the response as JSON and write it to the response writer
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}
}

func handleUCIInfo(w http.ResponseWriter, r *http.Request) {
	// Get the UCI engine information
	info := uciEngine.ID()
	options := uciEngine.Options()

	// Create a JSON response with the UCI engine information and options
	response := map[string]interface{}{
		"info":    info,
		"options": options,
	}

	// Set the content type to application/json
	w.Header().Set("Content-Type", "application/json")

	// Encode the response as JSON and write it to the response writer
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		return
	}
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	// Set the content type to text/html
	w.Header().Set("Content-Type", "text/html")

	// Open the play.html file
	file, err := os.Open("play.html")
	if err != nil {
		http.Error(w, "Failed to open play.html", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Copy the contents of the file to the response writer
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Failed to serve play.html", http.StatusInternalServerError)
		return
	}
}

func handleImg(w http.ResponseWriter, r *http.Request) {
	// Set the content type to image/png
	w.Header().Set("Content-Type", "image/png")

	// Extract the piece parameter from the query
	piece := r.URL.Query().Get("piece")
	if piece == "" {
		http.Error(w, "Piece not specified", http.StatusBadRequest)
		return
	}

	// Construct the file path for the requested piece image
	filePath := fmt.Sprintf("img/%s", piece)

	// Open the image file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open image file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Copy the contents of the image file to the response writer
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Failed to serve image", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", handlePlay)
	http.HandleFunc("/move", handleMove)
	http.HandleFunc("/show", handleShow)
	http.HandleFunc("/fen", handleFen)
	http.HandleFunc("/new", handleNewGame)
	http.HandleFunc("/info", handleUCIInfo)
	http.HandleFunc("/img", handleImg)

	server := &http.Server{Addr: config.ServerAddr}

	// Channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run the server in a goroutine
	go func() {
		log.Printf("Starting server on %s", config.ServerAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", config.ServerAddr, err)
		}
	}()

	// Block until a signal is received
	<-stop

	log.Println("Shutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt a graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := uciEngine.Close(); err != nil {
		log.Printf("Error quitting UCI engine: %v", err)
	}

	log.Println("Server exiting")
}
