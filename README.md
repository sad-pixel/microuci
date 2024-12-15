# MicroUCI Chess Development Server

MicroUCI Chess Development Server is a lightweight HTTP server / web application designed to aid in the development and testing of chess engines through a web interface. It supports UCI-compatible chess engines, such as Stockfish or Leela Chess Zero, and acts as a bridge between the web interface and the chess engine, allowing developers to test their engines directly from their browser.

## Installation

1. Clone the repository to your local environment.
2. Install a UCI-compatible chess engine (e.g., Stockfish, Leela Chess Zero) and ensure it is accessible via your system PATH.
3. Configure the server by creating a `config.yaml` file with your desired settings. Below is a sample configuration:
   ```yaml
    # Path to the UCI engine executable
    engine_path: "lc0"

    # Address and port for the server to listen on
    server_addr: ":8080"

    # Time allocated for the engine to calculate its move in milliseconds
    move_time: 100
    ```
4. Build the application
    ```bash
    go build
    ```
5. Run the server
    ```bash
    ./microuci
    ```
6. Navigate to `localhost:8080` in your browser.

## Configuration Format

The configuration for the MicroUCI Chess Development Server is specified in a `config.yaml` file. This file allows you to customize various aspects of the server and the chess engine it interfaces with. Below is a detailed description of the configuration options available:

- `engine_path`: Specifies the path to the UCI-compatible chess engine executable. This should be set to the location of your chess engine binary, such as "stockfish" or "lc0".

- `server_addr`: Defines the address and port on which the server will listen for incoming HTTP requests. The default value is ":8080", which means the server will listen on all available network interfaces on port 8080.

- `move_time`: Sets the time allocated for the chess engine to calculate its move, specified in milliseconds. 

- `uci_options`: A map of additional options that can be set for the UCI engine. These options are specific to the engine being used and can include settings such as:
  - `hash`: The size of the hash table in megabytes, which the engine uses for storing positions.
  - `ponder`: A boolean option to enable or disable pondering, which allows the engine to think during the opponent's turn.
  - `threads`: The number of CPU threads the engine should use for computation.
  - `WeightsFile`: Path to a weights file, which is specific to engines like Leela Chess Zero that use neural networks.

## API Reference

The MicroUCI Chess Development Server provides several HTTP endpoints to interact with the chess engine. Below is a list of available endpoints and their functionalities:

### `GET /new`

- **Description**: Initializes a new chess game.
- **Response**: Returns a JSON object with a message indicating that a new game has started.

### `GET /move`

- **Description**: Makes a move in the current game.
- **Parameters**:
  - `uci`: (optional) The move in UCI format.
  - `san`: (optional) The move in SAN format.
- **Response**: Returns a JSON object containing the best move calculated by the engine, the current FEN string, legal moves, engine info, and the PGN of the game.

### `GET /show`

- **Description**: Displays the current board position in ASCII format.
- **Response**: Returns a plain text representation of the current board position.

### `GET /fen`

- **Description**: Retrieves the current position in FEN format along with legal moves and PGN.
- **Response**: Returns a JSON object containing the FEN string, legal moves, and the PGN of the game.

### `GET /info`

- **Description**: Provides information about the UCI engine and its options.
- **Response**: Returns a JSON object with the engine's information and available options.

### `GET /img`

- **Description**: Retrieves an image of a chess piece.
- **Parameters**:
  - `piece`: The name of the piece to retrieve.
- **Response**: Returns a PNG image of the specified chess piece.
