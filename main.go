package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
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
	DoneChan        chan bool
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

func (g *Game) timer(stopChan chan bool) {
	ticker := NewTicker(time.Second)
	defer ticker.Stop()

	for remaining := time.Second * 40; remaining >= 0; remaining -= ticker.Duration() {
		select {
		case <-stopChan:
			ticker.Stop()
			return // Stop signal received, exit the timer
		case <-g.GameState.DoneChan:
			return // Game is done, exit the timer
		default:
			// Timer update logic
			MoveCursor(0, 0)
			fmt.Printf(Green+" TIMER: %v"+Reset, remaining)

			// Restore the user's cursor position
			MoveCursor(g.GameState.cursX, g.GameState.cursY)

			if remaining == 0 {
				// Timer expired, call gameOver
				g.gameOver()
				close(g.GameState.DoneChan) // Signal all goroutines to stop, corrected typo herep
				return
			}
		}
		ticker.Tick()
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

func readWrapper(inputChan chan byte, errorChan chan error, doneChan chan bool, game *Game) {
	for {
		select {
		case <-doneChan:
			return // Exit the goroutine if a done signal is received
		default:
			buf := make([]byte, 1) // Read one byte at a time
			n, err := os.Stdin.Read(buf)
			if err != nil {
				errorChan <- err
				return
			}
			if n > 0 {
				inputChan <- buf[0] // Send the byte to the channel
			}
		}
	}
}

func (g *Game) handleMainMenuInput(input string, inputChan chan byte, errorChan chan error, doneChan chan bool) {
	// Trim any whitespace from the input and make it lowercase
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "play":
		g.GameState.AppState = statePlaying
		g.startGame(inputChan, errorChan, doneChan)
	case "quit":
		g.GameState.AppState = stateQuit
		CursorShow()
		fmt.Println("Exiting the game. Goodbye!")
		time.Sleep(1 * time.Second)
		os.Exit(0)
	default:
		fmt.Println("Invalid choice, please try again.")
		g.displayMainMenu()

	}
}

func (g *Game) handleGameplayInput(input string, stopChan chan bool) {
	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "shit":
		g.processShitCommand()
	case "help":
		g.showHelp()
	case "quit":
		// Send a signal to stop the timer
		stopChan <- true
		safeClose(stopChan) // Safely close the channel

		// Handle 'quit' during gameplay
		g.cleanupGame() // Perform any necessary cleanup
		g.GameState.AppState = stateMainMenu
		g.displayMainMenu() // Display the main menu after quitting the game
		return
	default:
		fmt.Println("I don't understand:", input)
	}
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

func (g *Game) setupGameEnvironment() {
	// This function should set up the game environment (clear screen, display art, etc.)
}

func (g *Game) updateGameEnvironment() {
	// This function should update the game environment based on the current state
	// For example, display the main menu or the game screen
}

func (g *Game) cleanupGameEnvironment() {
	// This function should clean up the game environment when the game ends or quits
}

func (g *Game) cleanupGame() {
	// Close channels used during the game if they are no longer needed
	if g.GameState.DoneChan != nil {
		close(g.GameState.DoneChan)
		g.GameState.DoneChan = make(chan bool) // Reinitialize for future use
	}
}

func enableRawMode() (*unix.Termios, error) {
	originalState, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		return nil, err
	}

	newState := *originalState
	newState.Lflag &^= unix.ECHO   // Disable echo
	newState.Lflag &^= unix.ICANON // Disable canonical mode
	newState.Lflag &^= unix.ISIG   // Disable signal generation (Ctrl-C, Ctrl-Z)
	newState.Lflag &^= unix.IXON   // Disable XON/XOFF flow control

	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, &newState); err != nil {
		return nil, err
	}

	return originalState, nil
}

func disableRawMode(originalState *unix.Termios) error {
	return unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, originalState)
}

func (g *Game) run(inputChan chan byte, errorChan chan error, doneChan chan bool) {
	// Set up the game environment
	g.setupGameEnvironment()
	defer g.cleanupGameEnvironment()

	// Initialize the game state
	g.GameState.AppState = stateMainMenu

	g.displayMainMenu()

	g.GameState.cursX, g.GameState.cursY = 4, 23
	MoveCursor(g.GameState.cursX, g.GameState.cursY)

	// Enable raw mode
	originalState, err := enableRawMode()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to enable raw mode:", err)
		os.Exit(1)
	}
	defer disableRawMode(originalState) // Restore terminal state at the end

	var r []rune
	for g.GameState.AppState != stateQuit {
		select {
		case char := <-inputChan:
			if char == '\r' || char == '\n' {
				input := string(r)
				r = nil       // Reset buffer
				fmt.Println() // Move to the next line
				if g.GameState.AppState == stateMainMenu {
					g.handleMainMenuInput(input, inputChan, errorChan, doneChan)
				} else if g.GameState.AppState == statePlaying {
					g.handleGameplayInput(input, nil) // Pass nil if stopChan is not needed or not available
				}
			} else if char == '\b' || char == 127 {
				if len(r) > 0 {
					r = r[:len(r)-1]   // Remove the last character from the buffer
					fmt.Print("\b \b") // Handle backspace: move cursor back, print space, move cursor back again
				}
			} else {
				// Regular character input
				fmt.Print(string(char)) // Print character as it's typed
				r = append(r, rune(char))
				g.GameState.cursX++
				MoveCursor(g.GameState.cursX, g.GameState.cursY)
			}

		case err := <-errorChan:
			fmt.Println("Error reading input:", err)
			return
		}

		// Update the game environment based on the current state
		g.updateGameEnvironment()
	}
	close(doneChan) // Signal all goroutines to stop
}

func (g *Game) startGame(inputChan chan byte, errorChan chan error, doneChan chan bool) { // Clear the screen and display initial game art and messages
	g.GameState.AppState = statePlaying
	ClearScreen()
	displayAnsiFile(ArtFileDir + "2.ans")
	MoveCursor(4, 23)
	fmt.Print(Yellow + "You need to take a shit. Bad.")
	MoveCursor(4, 24) // Start from position 4 on the next line
	g.GameState.cursX, g.GameState.cursY = 4, 24

	stopChan := make(chan bool)

	go g.timer(stopChan)

	var r []rune
	for {
		select {
		case char := <-inputChan:
			runeChar := rune(char) // Convert byte to rune
			if runeChar == '\r' || runeChar == '\n' {
				input := string(r)
				r = nil // Reset buffer
				fmt.Println("\nInput received:", input)
				g.handleGameplayInput(input, stopChan) // Handle input with stopChan
				// Check if the state has changed to MainMenu, if so, break the loop
				if g.GameState.AppState == stateMainMenu {
					safeClose(stopChan) // Safely close the stop channel
					return
				}
			} else if runeChar == '\b' || runeChar == 127 {
				if len(r) > 0 {
					r = r[:len(r)-1] // Remove the last character from the buffer
					// Handle backspace for the terminal: Move cursor back, print space, move cursor back again
					fmt.Print("\b \b")
				}

			} else {
				fmt.Print(string(runeChar)) // Print character as it's typed
				r = append(r, runeChar)
				g.GameState.cursX++
				MoveCursor(g.GameState.cursX, g.GameState.cursY)

			}

		case err := <-errorChan:
			fmt.Println("Error reading input:", err)
			safeClose(stopChan) // Safely close the stop channel
			// Cleanup and exit the game
			return

		}
		// Check if the state has changed to MainMenu, if so, break the loop
		if g.GameState.AppState == stateMainMenu {
			return
		}
	}
}

// Safe channel close utility function
func safeClose(ch chan bool) {
	select {
	case <-ch:
		// Channel already closed
	default:
		close(ch)
	}
}

func (g *Game) processInputDuringGameplay(char byte, r *[]rune, stopChan chan bool) {
	// Process each character received during gameplay
	if char == '\r' || char == '\n' {
		// Handle enter key
		input := string(*r)
		*r = nil // Reset buffer
		fmt.Println("\nInput received:", input)
		g.handleGameplayInput(input, stopChan)
	} else if char == '\b' || char == 127 {
		// Handle backspace
		if len(*r) > 0 {
			*r = (*r)[:len(*r)-1] // Remove the last character from the buffer
			// Handle backspace for the terminal: Move cursor back, print space, move cursor back
			fmt.Print("\b \b")
		}
	} else {
		// Handle normal characters
		fmt.Print(string(char)) // Print character as it's typed
		*r = append(*r, rune(char))
	}
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
		AppState:        stateMainMenu,
		DoneChan:        make(chan bool),
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

func main() {
	// Define the flags
	localDisplayPtr := flag.Bool("local", false, "use local UTF-8 display instead of CP437")
	pathPtr := flag.String("path", "", "path to door32.sys file (optional if --local is set)")

	// Parse the flags
	flag.Parse()

	// Use the flag values
	localDisplay := *localDisplayPtr

	// Initialize the game
	game := initializeGame(localDisplay, *pathPtr)

	// Input channels
	inputChan := make(chan byte)
	errorChan := make(chan error)
	doneChan := make(chan bool)

	// Start the input reading goroutine
	go readWrapper(inputChan, errorChan, doneChan, game)

	// Start the game
	game.run(inputChan, errorChan, doneChan)
}
