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
	LastAppState    int
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

func (g *Game) setupGameEnvironment() {
	// This function should set up the game environment (clear screen, display art, etc.)
	ClearScreen()
	displayAnsiFile(ArtFileDir + "main.ans")
	MoveCursor(1, 1)
	fmt.Printf(WhiteHi+" Welcome,%s"+Reset, g.User.Alias)
	// MoveCursor(6, 24)
}

func (g *Game) updateGameEnvironment() {
	// Only update the environment if the game state has changed
	if g.GameState.AppState != g.GameState.LastAppState {
		switch g.GameState.AppState {
		case stateMainMenu:
			ClearScreen()
			displayAnsiFile(ArtFileDir + "main.ans")
			MoveCursor(1, 1)
			fmt.Printf(WhiteHi+" Welcome,%s"+Reset, g.User.Alias)
			g.GameState.cursX, g.GameState.cursY = 7, 23
			MoveCursor(7, 23)

		case statePlaying:
			ClearScreen()
			displayAnsiFile(ArtFileDir + "start.ans")
			MoveCursor(2, 23)
			fmt.Print(BgBlue + CyanHi + "You need to take a shit. Bad." + Reset)
			MoveCursor(5, 24) // Start from position 4 on the next line
			fmt.Print(YellowHi)
			g.GameState.cursX, g.GameState.cursY = 5, 24

			// ... other cases ...
		}
		g.GameState.LastAppState = g.GameState.AppState
	}
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
func (g *Game) handleMainMenuInput(input string, inputChan chan byte, errorChan chan error, doneChan chan bool, resetTimerChan chan bool) { // Trim any whitespace from the input and make it lowercase
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "play":
		g.GameState.AppState = statePlaying
		g.startGame(inputChan, errorChan, doneChan, resetTimerChan)
	case "quit":
		g.GameState.AppState = stateQuit
		CursorHide()
		MoveCursor(7, 23)
		fmt.Println(BgBlue + RedHi + "Exiting the game. Goodbye!" + Reset)
		time.Sleep(1 * time.Second)
		fmt.Print(BgBlue + RedHi + "                         " + Reset)
		MoveCursor(7, 23)
		CursorShow()
		os.Exit(0)
	default:
		CursorHide()
		MoveCursor(7, 23)
		fmt.Print(BgBlue + RedHi + "Invalid choice!" + Reset)
		time.Sleep(1 * time.Second)
		MoveCursor(7, 23)
		fmt.Print(BgBlue + RedHi + "                " + Reset)
		MoveCursor(7, 23)
		g.GameState.cursX, g.GameState.cursY = 7, 23
		g.updateGameEnvironment() // Use this to display the correct environment based on the current state
		CursorShow()
		resetTimerChan <- true // Reset the idle timer
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
		g.updateGameEnvironment() // Properly load the main menu environment
		return
	default:
		MoveCursor(2, 23)
		fmt.Print(BgBlue + RedHi + "                                                                        " + Reset)
		MoveCursor(2, 23)
		fmt.Fprintf(os.Stdout, BgBlue+CyanHi+"I don't know how to "+RedHi+"%s"+Reset, input)
		MoveCursor(5, 24)
		fmt.Print(BgBlue + RedHi + "                                                                        " + Reset)
		g.GameState.cursX, g.GameState.cursY = 5, 24

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

func (g *Game) gameOver() {
	// Display a game over message or perform other necessary actions
	fmt.Println("Game Over! Time's up.")
	time.Sleep(2 * time.Second)
	// g.displayMainMenu()
}

func (g *Game) startGame(inputChan chan byte, errorChan chan error, doneChan chan bool, resetTimerChan chan bool) {
	g.GameState.AppState = statePlaying
	g.updateGameEnvironment()
	stopChan := make(chan bool)

	go g.timer(stopChan)

	fmt.Print(Reset)

	var r []rune
	for {
		fmt.Print(BgBlue + YellowHi)
		select {
		case char := <-inputChan:
			runeChar := rune(char) // Convert byte to rune
			if runeChar == '\r' || runeChar == '\n' {
				input := string(r)
				r = nil // Reset buffer
				fmt.Print(Reset)

				// fmt.Println("\nInput received:", input)
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
					g.GameState.cursX--
				}

			} else {
				fmt.Print(string(runeChar)) // Print character as it's typed
				r = append(r, runeChar)
				g.GameState.cursX++
				MoveCursor(g.GameState.cursX, g.GameState.cursY)
				resetTimerChan <- true // Reset the idle timer
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
		fmt.Print(Reset)
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

func (g *Game) run(inputChan chan byte, errorChan chan error, doneChan chan bool, resetTimerChan chan bool) {
	// Set up the game environment

	g.GameState.AppState = stateMainMenu
	g.setupGameEnvironment()
	defer g.cleanupGameEnvironment()

	MoveCursor(7, 23) // Start from position 4 on the next line

	fmt.Print(BgBlue + YellowHi)

	// Enable raw mode
	originalState, err := enableRawMode()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to enable raw mode:", err)
		os.Exit(1)
	}
	defer disableRawMode(originalState) // Restore terminal state at the end

	var r []rune
	for g.GameState.AppState != stateQuit {
		fmt.Print(BgBlue + YellowHi)
		select {
		case char := <-inputChan:
			resetTimerChan <- true // Reset the idle timer
			if char == '\r' || char == '\n' {
				input := string(r)
				r = nil          // Reset buffer
				fmt.Print(Reset) // Move to the next line
				if g.GameState.AppState == stateMainMenu {
					g.handleMainMenuInput(input, inputChan, errorChan, doneChan, resetTimerChan)
				} else if g.GameState.AppState == statePlaying {
					g.handleGameplayInput(input, nil) // Pass nil if stopChan is not needed or not available
				}
			} else if char == '\b' || char == 127 {
				if len(r) > 0 {
					r = r[:len(r)-1]   // Remove the last character from the buffer
					fmt.Print("\b \b") // Handle backspace: move cursor back, print space, move cursor back again
					g.GameState.cursX--
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
		cursX:           7,
		cursY:           23,
		AppState:        stateMainMenu,
		DoneChan:        make(chan bool),
		LastAppState:    stateMainMenu,
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
	resetTimerChan := make(chan bool)

	// Start the input reading goroutine
	go readWrapper(inputChan, errorChan, doneChan, game)

	// Start the idle timer goroutine
	go func() {
		// Set the idle timeout duration to 3 minutes
		idleTimeout := 3 * time.Minute
		idleTimer := time.NewTimer(idleTimeout)

		for {
			select {
			case <-idleTimer.C:
				// Idle timeout reached, print the message and exit
				fmt.Println("Idle timeout -- come back another time!")
				os.Exit(0)
			case <-resetTimerChan:
				// Keyboard input received, reset the timer
				if !idleTimer.Stop() {
					<-idleTimer.C
				}
				idleTimer.Reset(idleTimeout)
			}
		}
	}()

	// Start the game
	game.run(inputChan, errorChan, doneChan, resetTimerChan)
}
