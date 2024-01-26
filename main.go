package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	ArtFileDir = "art/"
	ansi       = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
)

type User struct {
	Alias     string
	TimeLeft  time.Duration
	Emulation int
	NodeNum   int
	H         int
	W         int
	ModalH    int
	ModalW    int
}

type Game struct {
	User      User
	GameState GameState
	Awards    map[string]bool
}

type GameState struct {
	Turn            int
	OpenDoor        bool
	RemovePants     bool
	EnterToilet     bool
	SitDown         bool
	TakePill        bool
	FartLightly     bool
	stopTime        bool
	pressureMessage bool
	cursX           int
	cursY           int
}

type Ticker interface {
	Duration() time.Duration
	Tick()
	Stop()
}

type TickFunc func(d time.Duration)

type ticker struct {
	*time.Ticker
	d time.Duration
}

func (t *ticker) Tick()                   { <-t.C }
func (t *ticker) Duration() time.Duration { return t.d }

func NewTicker(d time.Duration) Ticker {
	return &ticker{time.NewTicker(d), d}
}

func readWrapper(dataChan chan []byte, errorChan chan error) {
	for {
		buf := make([]byte, 1) // Read one byte at a time
		n, err := os.Stdin.Read(buf)
		if err != nil {
			errorChan <- err
			return
		}
		if n > 0 {
			os.Stdout.Write(buf[:n]) // Echo the input back to the user
			dataChan <- buf[:n]
		}
	}
}

func (g *Game) Countdown(ticker Ticker, duration time.Duration, stopChan <-chan bool) chan time.Duration {
	remainingCh := make(chan time.Duration, 1)
	go func() {
		for remaining := duration; remaining >= 0; remaining -= ticker.Duration() {
			select {
			case <-stopChan:
				// Handle stop signal
				ticker.Stop()
				return
			default:
				// Countdown logic
				if remaining < time.Second*20 {
					// Specific logic when the timer is under 20 seconds
					if g.GameState.pressureMessage {
						MoveCursor(4, 22)
						fmt.Print(EraseLine)
						fmt.Println("Hurry! You need to find a way to reduce the pressure in your gut.")
						g.GameState.pressureMessage = false
					}
				}
				remainingCh <- remaining
				ticker.Tick()
			}
		}
		ticker.Stop()
		close(remainingCh)
		MoveCursor(0, 0)
		fmt.Print(Red)
		fmt.Print(" TIME'S UP!")
		fmt.Print(Reset)
	}()
	return remainingCh
}

func initializeGame(localDisplay bool, dropPath string) *Game {
	// Initialize the User with either default values or based on command-line arguments

	var user User
	if localDisplay {
		// Set default values when --local is used
		user = User{
			Alias:     "SysOp",
			TimeLeft:  120 * time.Minute,
			Emulation: 1,
			NodeNum:   1,
			H:         25,
			W:         80,
			ModalH:    25,
			ModalW:    80,
		}
	} else {
		// Check for required --path argument if --local is not set
		if dropPath == "" {
			fmt.Fprintln(os.Stderr, "missing required -path argument")
			os.Exit(2)
		}

		user = Initialize(dropPath)

	}

	// Initialize GameState with default or initial values
	gameState := GameState{
		Turn:            1,
		OpenDoor:        false,
		RemovePants:     false,
		EnterToilet:     false,
		SitDown:         false,
		TakePill:        false,
		FartLightly:     false,
		stopTime:        false,
		pressureMessage: false,
		cursX:           4,
		cursY:           23,
	}

	// Initialize a map for tracking awards
	awards := make(map[string]bool)

	// Initialize the Game struct with the components
	game := &Game{
		User:      user,
		GameState: gameState,
		Awards:    awards,
	}

	return game
}

func (g *Game) run() {
	errorChan := make(chan error)
	dataChan := make(chan []byte)

	// Display the main menu
	g.displayMainMenu()

	// Start reading input asynchronously
	go readWrapper(dataChan, errorChan)
	r := bytes.NewBuffer(nil)

	for {
		select {
		case data := <-dataChan:
			for _, char := range data {
				switch char {
				case '\r', '\n':
					// Handle enter key
					input := r.String()
					r.Reset()
					g.processMainMenuInput(input)
				default:
					// Accumulate characters in buffer
					r.WriteByte(char)
				}
			}
		case err := <-errorChan:
			// Handle any read errors
			fmt.Println("Error reading input:", err)
			return
		}
	}
}

func (g *Game) processMainMenuInput(input string) {
	switch strings.ToLower(input) {
	case "play":
		g.startGame()
	case "awards":
		g.showAwards()
	case "quit":
		fmt.Println("Quitting...")
		time.Sleep(1 * time.Second)
		CursorShow()
		os.Exit(0)
	default:
		fmt.Println("Invalid choice, please try again.")
		g.displayMainMenu()
	}
}

func (g *Game) handleInput(input string) {
	// Trim any whitespace from the input
	input = strings.TrimSpace(input)

	// Handle different cases based on the input
	switch input {
	case "shit":
		// Example case if the user types "shit"
		g.processShitCommand()
	case "help":
		// Example case if the user asks for help
		g.showHelp()
	case "quit":
		g.run()
	default:
		// Handle unknown commands
		fmt.Println("I don't know how to ", input)
	}

	// You can add more cases here depending on the commands
	// you want to handle in your game.
}

func (g *Game) processShitCommand() {
	// Implement what happens when the user types "shit"
	fmt.Println("Processing 'shit' command...")
	// Example: Update GameState, trigger events, etc.
}

func (g *Game) showHelp() {
	// Implement help instructions
	fmt.Println("Help instructions go here...")
	// Example: Display commands and descriptions
}

func (g *Game) timer(stopChan chan bool) {
	ticker := NewTicker(time.Second)
	defer ticker.Stop()

	// Save the user's cursor position
	// userCursorX, userCursorY := g.GameState.cursX, g.GameState.cursY

	for remaining := time.Second * 40; remaining >= 0; remaining -= ticker.Duration() {
		select {
		case <-stopChan:
			return // Stop signal received, exit the timer
		default:
			// Timer update logic
			// MoveCursor(0, 0)
			// fmt.Printf(Green+" TIMER: %v"+Reset, remaining)

			// // Restore the user's cursor position
			// MoveCursor(userCursorX, userCursorY)

			if remaining == 0 {
				// Timer expired, call gameOver
				g.gameOver()
				return
			}
		}
		ticker.Tick()
	}
}

func (g *Game) startGame() {
	// Clear the screen and display initial game art and messages
	ClearScreen()
	displayAnsiFile(ArtFileDir + "2.ans")
	MoveCursor(4, 23)
	fmt.Print(Yellow + "You need to take a shit. Bad.")
	MoveCursor(4, 24) // Start from position 4 on the next line
	g.GameState.cursX, g.GameState.cursY = 4, 24

	// Initialize channels for timer and input handling
	stopChan := make(chan bool)
	errorChan := make(chan error)
	dataChan := make(chan []byte)

	// Start the timer
	go g.timer(stopChan)

	// Set terminal to raw mode for input handling
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Error setting terminal to raw mode:", err)
		return
	}

	// Ensure terminal is restored to non-raw mode when function exits
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			fmt.Println("Error restoring terminal state:", err)
		}
	}()

	// Start reading input asynchronously
	go readWrapper(dataChan, errorChan)

	// Buffer for handling rune-based input
	var r []rune

	// Main input handling loop
	for {
		select {
		case data := <-dataChan:
			for _, char := range data {
				switch char {
				case '\b', 127: // Backspace handling
					if len(r) > 0 {
						r = r[:len(r)-1]
						if g.GameState.cursX > 4 {
							g.GameState.cursX--
							fmt.Print("\b \b") // Move cursor back and clear character
							os.Stdout.Sync()   // Flush output buffer
						}
					}
				case '\r', '\n': // Enter key handling
					input := string(r)
					r = nil                                 // Reset buffer
					fmt.Println("\nInput received:", input) // Debugging: print input

					if strings.ToLower(input) == "quit" {
						stopChan <- true
						close(stopChan)
						return // Exit function, triggering defer to restore terminal state
					}

					g.handleInput(input) // Handle other inputs

				default: // Regular character handling
					r = append(r, rune(char))
					g.GameState.cursX++
					MoveCursor(g.GameState.cursX, g.GameState.cursY)
					fmt.Printf("%c", char)
					os.Stdout.Sync()
				}
			}

		case err := <-errorChan: // Handle any read errors
			fmt.Println("Error reading input:", err)
			return // Exit function, triggering defer to restore terminal state
		}

		// Additional game logic here (if any)
	}
	// Exiting the loop will trigger the defer statement to restore terminal state
}

func (g *Game) showAwards() {
	fmt.Println("Displaying Awards...")
}

func (g *Game) displayMainMenu() {
	ClearScreen()
	displayAnsiFile(ArtFileDir + "main.ans")
	MoveCursor(0, 0)
	fmt.Printf(WhiteHi+" Welcome, %s"+Reset, g.User.Alias)
}

func (g *Game) gameOver() {
	ClearScreen()
	// Display a game over message or perform other necessary actions
	fmt.Println("Game Over! Time's up.")
	time.Sleep(2 * time.Second)
	g.displayMainMenu()
}

func main() {
	// Define the flags
	localDisplayPtr := flag.Bool("local", false, "use local UTF-8 display instead of CP437")
	pathPtr := flag.String("path", "", "path to door32.sys file (optional if --local is set)")

	// Parse the flags
	flag.Parse()

	// Use the flag values
	localDisplay = *localDisplayPtr

	// Initialize and run the game
	game := initializeGame(localDisplay, *pathPtr)
	game.run()
}
