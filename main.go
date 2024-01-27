package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	ArtFileDir    = "art/"
	ansi          = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"
	stateMainMenu = iota
	statePlaying
	stateQuit
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
	AppState        int
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

	for remaining := time.Second * 40; remaining >= 0; remaining -= ticker.Duration() {
		select {
		case <-stopChan:
			return // Stop signal received, exit the timer
		default:
			// Timer update logic
			MoveCursor(0, 0)
			fmt.Printf(Green+" TIMER: %v"+Reset, remaining)

			// Restore the user's cursor position
			MoveCursor(g.GameState.cursX, g.GameState.cursY)

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

	stopChan := make(chan bool)
	errorChan := make(chan error)
	dataChan := make(chan []byte)

	go g.timer(stopChan)
	go readWrapper(dataChan, errorChan)

	// Save the current terminal settings
	saveCmd := exec.Command("/bin/stty", "-F", "/dev/tty", "-g")
	saveOut, err := saveCmd.Output()
	if err != nil {
		fmt.Println("Failed to get current stty settings:", err)
		return
	}
	originalSettings := strings.TrimSpace(string(saveOut))

	// Set terminal to raw mode
	setRawCmd := exec.Command("/bin/stty", "-F", "/dev/tty", "-icanon", "min", "1")
	err = setRawCmd.Run()
	if err != nil {
		fmt.Println("Failed to set raw mode:", err)
		return
	}

	// Schedule the terminal restore code to run when the function exits
	defer func() {
		restoreCmd := exec.Command("/bin/stty", "-F", "/dev/tty", originalSettings)
		err = restoreCmd.Run()
		if err != nil {
			fmt.Println("Failed to restore stty settings:", err)
		}
	}()

	var r []rune

	for {
		select {
		case data := <-dataChan:
			char := rune(data[0])

			if char == '\r' || char == '\n' {
				input := string(r)
				r = nil                                 // Reset buffer
				fmt.Println("\nInput received:", input) // Debugging: print input

				// Special handling for "quit" command
				if strings.ToLower(input) == "quit" {
					// Stop the timer
					stopChan <- true
					close(stopChan)

					// Handle the "quit" input
					g.handleInput(input)

					return // Exit startGame function
				}
			} else if char == '\b' || char == 127 {
				if len(r) > 0 {
					r = r[:len(r)-1] // Remove the last character from the buffer
				}
			} else {
				// Regular character handling
				// fmt.Printf("%c", char) // Echo the character
				g.GameState.cursX++
				MoveCursor(g.GameState.cursX, g.GameState.cursY)
				r = append(r, char)
			}

		case err := <-errorChan:
			fmt.Println("Error reading input:", err)
			return
		}
	}
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

func (g *Game) processMainMenuInput(input string) {
	switch strings.ToLower(input) {
	case "play":
		g.GameState.AppState = statePlaying
	case "quit":
		g.GameState.AppState = stateQuit
	default:
		fmt.Println("Invalid choice, please try again.")
		g.displayMainMenu()
	}
}

func (g *Game) run() {
	errorChan := make(chan error)
	dataChan := make(chan []byte)

	g.GameState.AppState = stateMainMenu // Initialize the app state

	for g.GameState.AppState != stateQuit {
		switch g.GameState.AppState {
		case stateMainMenu:
			// Display the main menu
			g.displayMainMenu()

			// Start reading input asynchronously
			go readWrapper(dataChan, errorChan)
			r := bytes.NewBuffer(nil)

			for g.GameState.AppState == stateMainMenu {
				select {
				case data := <-dataChan:
					for _, char := range data {
						switch char {
						case '\r', '\n':
							// Handle enter key
							input := r.String()
							r.Reset()
							g.processMainMenuInput(input) // This will update g.GameState.AppState
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

		case statePlaying:
			// Start the game
			g.startGame()
			g.GameState.AppState = stateMainMenu // After the game ends, return to the main menu

			// Add more states as needed
		}
	}
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
