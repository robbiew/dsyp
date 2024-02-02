package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// Game constants
const (
	ArtFileDir    = "art/"
	stateMainMenu = iota
	statePlaying
	stateQuit
	stateGameOver
	stateHelp
	stateCredits
	stateIntro
	stateAwards
	LogLevelInput = iota
	LogLevelWarning
	LogLevelError
)

// Verb lists
var (
	lookVerbs      = []string{"look", "check", "examine"}
	openVerbs      = []string{"open", "push"}
	breakVerbs     = []string{"break", "smash"}
	pullVerbs      = []string{"pull", "yank"}
	closeVerbs     = []string{"close", "shut", "slam"}
	removeVerbs    = []string{"remove", "drop", "off"}
	wearVerbs      = []string{"wear", "on"}
	moveVerbs      = []string{"move", "enter", "go"}
	poopVerbs      = []string{"poop", "poo", "crap", "dump", "shit", "defecate"}
	eatVerbs       = []string{"eat", "take"}
	dieVerbs       = []string{"die", "kill", "suicide"}
	lightlyAdverbs = []string{"lightly", "light", "gently", "softly", "soft", "little", "small", "tiny"}
)

// Noun lists
var (
	bathroomNouns = []string{"bathroom", "washroom", "restroom"}
	pillsNouns    = []string{"pills", "pill", "drugs"}
)

// Define lists of words
var lists = map[string][]string{
	"lookVerbs":      lookVerbs,
	"openVerbs":      openVerbs,
	"breakVerbs":     breakVerbs,
	"pullVerbs":      pullVerbs,
	"closeVerbs":     closeVerbs,
	"removeVerbs":    removeVerbs,
	"wearVerbs":      wearVerbs,
	"moveVerbs":      moveVerbs,
	"poopVerbs":      poopVerbs,
	"eatVerbs":       eatVerbs,
	"dieVerbs":       dieVerbs,
	"lightlyAdverbs": lightlyAdverbs,
}

// Define a map to associate each list of words with its corresponding "Main Word"
var mainWordMappings = map[string]string{
	"lookVerbs":      "look",
	"openVerbs":      "open",
	"breakVerbs":     "break",
	"pullVerbs":      "pull",
	"closeVerbs":     "close",
	"removeVerbs":    "remove",
	"wearVerbs":      "wear",
	"moveVerbs":      "move",
	"poopVerbs":      "shit",
	"eatVerbs":       "eat",
	"dieVerbs":       "die",
	"lightlyAdverbs": "lightly",
	"bathroomNouns":  "bathroom",
	"pillsNouns":     "pills",
}

type User struct {
	Alias        string
	TimeLeft     time.Duration
	Emulation    int
	NodeNum      int
	H            int
	W            int
	ModalH       int
	ModalW       int
	LocalDisplay bool
	Awards       map[string]bool
}

type Game struct {
	User            User
	GameState       GameState
	Awards          map[string]bool
	AwardedAwards   map[string]bool
	mutex           sync.Mutex
	UserInputBuffer []string
}

type GameState struct {
	Door         bool
	Pants        bool
	Standing     bool
	Farts        int
	Pills        bool
	PillTimer    time.Duration
	stopTime     bool
	cursX        int
	cursY        int
	AppState     int
	DoneChan     chan bool
	LastAppState int
	OnMainMenu   bool
}

func inputLog(level int, userAlias string, message string) {
	switch level {
	case LogLevelInput:
		formattedMessage := fmt.Sprintf("[INPUT] [%s]: %s", userAlias, message)
		log.Print(formattedMessage)
	case LogLevelWarning:
		log.Print("[WARNING] " + message)
	case LogLevelError:
		log.Print("[ERROR] " + message)
	}
}

func enableRawMode() (*unix.Termios, error) {
	originalState, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	if err != nil {
		inputLog(LogLevelError, "SysOp", "Failed to get terminal state")
		return nil, err
	}

	newState := *originalState
	newState.Lflag &^= unix.ECHO   // Disable echo
	newState.Lflag &^= unix.ICANON // Disable canonical mode
	newState.Lflag &^= unix.ISIG   // Disable signal generation (Ctrl-C, Ctrl-Z)
	newState.Lflag &^= unix.IXON   // Disable XON/XOFF flow control

	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, &newState); err != nil {
		inputLog(LogLevelError, "SysOp", "Failed to set terminal state")
		return nil, err
	}

	return originalState, nil
}

func disableRawMode(originalState *unix.Termios) error {
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TCSETS, originalState); err != nil {
		inputLog(LogLevelError, "SysOp", "Failed to restore terminal state")
		return err
	}
	return nil
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
				inputLog(LogLevelError, "SysOp", "Failed to read input")
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
	displayAnsiFile(ArtFileDir+"main.ans", g.User.LocalDisplay)
	MoveCursor(4, 2)
	fmt.Printf(BgMagenta+YellowHi+"%s"+WhiteHi+":"+Reset, g.User.Alias)
	// MoveCursor(6, 24)
}

func (g *Game) updateGameEnvironment() {
	// Only update the environment if the game state has changed
	if g.GameState.AppState != g.GameState.LastAppState {
		ClearScreen()

		switch g.GameState.AppState {

		case stateMainMenu:
			g.GameState.OnMainMenu = true
			displayAnsiFile(ArtFileDir+"main.ans", g.User.LocalDisplay)
			MoveCursor(4, 2)
			fmt.Printf(BgMagenta+YellowHi+"%s"+WhiteHi+":"+Reset, g.User.Alias)
			g.GameState.cursX, g.GameState.cursY = 7, 23
			MoveCursor(7, 23)
			fmt.Print(Reset)

		case statePlaying:
			g.GameState.OnMainMenu = false
			displayAnsiFile(ArtFileDir+"start.ans", g.User.LocalDisplay)
			MoveCursor(2, 23)
			fmt.Print(BgBlue + CyanHi + "You need to take a shit. Bad." + Reset)
			MoveCursor(5, 24)
			fmt.Print(YellowHi)
			g.GameState.cursX, g.GameState.cursY = 5, 24
			fmt.Print(Reset)

		case stateGameOver:
			g.GameState.OnMainMenu = false
			fmt.Print(Reset)
			g.GameState.AppState = stateMainMenu
			g.setupGameEnvironment()
			g.GameState.cursX, g.GameState.cursY = 7, 23
			MoveCursor(7, 23)

		case stateIntro:
			g.GameState.OnMainMenu = false
			ClearScreen()
			CursorHide()
			displayAnsiFile(ArtFileDir+"intro.ans", g.User.LocalDisplay)

			PrintStringLoc("3", 40, 19)
			DelayedAction(1*time.Second, func() {
				PrintStringLoc("2", 40, 19)
			})

			DelayedAction(1*time.Second, func() {
				PrintStringLoc("1", 40, 19)
			})

			DelayedAction(1*time.Second, func() {
				PrintStringLoc("GO!", 39, 19)
			})

			DelayedAction(1*time.Second, func() {
				fmt.Print(Reset)
				CursorShow()
			})

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

func DelayedAction(duration time.Duration, action func()) {
	done := make(chan bool)

	go func() {
		time.Sleep(duration)
		action() // Execute the specified action
		done <- true
	}()

	<-done // Wait for the goroutine to complete
}

func (g *Game) handleMainMenuInput(input string, inputChan chan byte, errorChan chan error, doneChan chan bool) {
	if input != "" {
		inputLog(LogLevelInput, g.User.Alias, input)
	}

	// Check if the input matches any verb from poopVerbs
	for _, verb := range poopVerbs {
		if input == verb {
			// If a match is found, add the "main" word to the user's input buffer
			g.UserInputBuffer = append(g.UserInputBuffer, "shit")
			break
		}
	}

	switch input {
	case "play":
		g.GameState.AppState = statePlaying
		g.startGame(inputChan, errorChan, doneChan)
	case "quit", "exit":
		g.GameState.AppState = stateQuit
		CursorHide()
		MoveCursor(7, 23)
		fmt.Println(BgBlue + RedHi + "Exiting the game. Goodbye!" + Reset)
		DelayedAction(2*time.Second, func() {
			CursorShow()
		})
	case "awards":
		g.GameState.AppState = stateAwards
		g.updateGameEnvironment()

		ClearScreen()
		CursorHide()

		// Check if the user has earned any awards
		if len(g.User.Awards) > 0 {
			// Display the user's awards
			fmt.Println("Awards earned by", g.User.Alias+":")
			for awardID, earned := range g.User.Awards {
				if earned {
					// Print the name of the award
					awardName := getAwardNameByID(awardID)
					fmt.Println("- " + awardName)
				}
			}
		} else {
			// User has no awards
			fmt.Println("No awards earned yet by", g.User.Alias)
		}

		// Wait for a single keypress
		g.readSingleKeyPress(inputChan, stateMainMenu)
		g.GameState.AppState = stateMainMenu
		MoveCursor(7, 23)
		CursorShow()
	default:
		// Check if the input matches any verb from poopVerbs
		for _, verb := range poopVerbs {
			if input == verb {
				g.mutex.Lock()
				defer g.mutex.Unlock()
				g.processShitCommand(inputChan)
				g.GameState.AppState = stateMainMenu
				g.updateGameEnvironment()
				return
			}
		}

		// If no matching verb is found, handle it as an invalid choice
		CursorHide()
		MoveCursor(7, 23)
		fmt.Print(BgBlue + RedHi + "Invalid choice!" + Reset)

		DelayedAction(1*time.Second, func() {
			MoveCursor(7, 23)
			fmt.Print(BgBlue + RedHi + "                       " + Reset)
			MoveCursor(7, 23)
			g.GameState.cursX, g.GameState.cursY = 7, 23
			g.updateGameEnvironment()
			CursorShow()
		})
	}
}

func (g *Game) handleGameplayInput(input string, stopChan chan bool, inputChan chan byte) {
	if input != "" {
		inputLog(LogLevelInput, g.User.Alias, input)
	}

	// Check if the input matches any of the "Main Words" from the mappings
	if mainWord, ok := mainWordMappings[input]; ok {
		// If a match is found, add the "Main Word" to the user's input buffer
		g.UserInputBuffer = append(g.UserInputBuffer, mainWord)
	}

	switch input {
	case "quit":
		// Similar to "shit," protect any shared resources with a mutex
		g.mutex.Lock()
		defer g.mutex.Unlock()
		// Send a signal to stop the timer
		stopChan <- true
		safeClose(stopChan) // Safely close the channel
		g.cleanupGame()     // Perform any necessary cleanup
		g.GameState.AppState = stateGameOver
		g.updateGameEnvironment()
		return
	default:
		// Check if the input matches any verb from poopVerbs
		for _, verb := range poopVerbs {
			if input == verb {
				g.mutex.Lock()
				defer g.mutex.Unlock()
				stopChan <- true
				safeClose(stopChan) // Safely close the channel
				g.processShitCommand(inputChan)
				g.GameState.AppState = stateMainMenu
				g.updateGameEnvironment()
				return
			}
		}

		// If no matching verb is found, handle it as an invalid choice
		CursorHide()
		MoveCursor(2, 23)
		fmt.Print(BgBlue + RedHi + "                                                                             " + Reset)
		MoveCursor(2, 23)
		fmt.Print(BgBlue + RedHi + "I don't know how to " + Reset + BgBlue + CyanHi + input + Reset)

		MoveCursor(5, 24)
		fmt.Print(BgBlue + RedHi + "                                                                          " + Reset)
		MoveCursor(5, 24)
		g.GameState.cursX, g.GameState.cursY = 5, 24
		CursorShow()
	}
}

func (g *Game) readSingleKeyPress(inputChan chan byte, nextState int) {
	// Wait for a single keypress

	CursorHide()
	MoveCursor(0, 23)
	CenterText("Press a Key to Continue", 80)

	<-inputChan

	// Transition to the nextState immediately upon any keypress
	CursorShow()
	g.GameState.AppState = nextState
	g.updateGameEnvironment()
}

func (g *Game) processShitCommand(inputChan chan byte) {
	// Check and grant any awards
	g.UserInputBuffer = append(g.UserInputBuffer, "shit")
	g.checkAndGrantAwards(inputChan)
	ClearScreen()

	// Display user's awarded awards
	awardsEarned := false
	for _, award := range awards {
		if g.User.Awards[award.ID] {
			if !awardsEarned {
				fmt.Println("Awards earned by", g.User.Alias+":")
				awardsEarned = true
			}
			awardName := getAwardNameByID(award.ID)
			fmt.Println(awardName)
		}
	}

	if !awardsEarned {
		fmt.Println("You shit your pants!")
	}

	// Pause for a keypress
	g.readSingleKeyPress(inputChan, stateMainMenu)

	// Clear the input buffer here
	g.UserInputBuffer = []string{}
	g.GameState.AppState = stateAwards
	g.updateGameEnvironment()
}

func (g *Game) startGame(inputChan chan byte, errorChan chan error, doneChan chan bool) {
	g.GameState.AppState = stateIntro
	g.updateGameEnvironment()

	g.GameState.AppState = statePlaying
	g.updateGameEnvironment()

	stopChan := make(chan bool)

	go g.timer(stopChan)

	var r []rune
	for {
		fmt.Print(BgBlue + YellowHi)
		select {
		case char := <-inputChan:
			runeChar := rune(char) // Convert byte to rune
			if runeChar == '\r' || runeChar == '\n' {

				input := sanitizeInput(strings.ToLower(string(r)))
				r = nil // Reset buffer
				fmt.Print(Reset)

				// fmt.Println("\nInput received:", input)
				g.handleGameplayInput(input, stopChan, inputChan) // Handle input with stopChan

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

			}

		case err := <-errorChan:
			fmt.Println("Error reading input:", err)
			log.Print("Error reading input:", err)
			safeClose(stopChan) // Safely close the stop channel
			// Cleanup and exit the game
			return

		}
		// Check if the state has changed to MainMenu, if so, break the loop
		if g.GameState.AppState != statePlaying {
			return
		}
		fmt.Print(Reset)
	}
}

func sanitizeInput(input string) string {
	// Remove leading and trailing spaces
	input = strings.TrimSpace(input)

	// Remove any invalid characters or perform additional sanitization if needed
	// For example, you can use a regular expression to allow only specific characters

	// Example: Allow only letters, digits, and spaces
	validChars := regexp.MustCompile(`[^a-zA-Z0-9 ]`)
	input = validChars.ReplaceAllString(input, "")

	return input
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

func (g *Game) run(inputChan chan byte, errorChan chan error, doneChan chan bool) {
	// Set up the game environment
	g.GameState.AppState = stateMainMenu
	g.setupGameEnvironment()
	defer g.cleanupGameEnvironment()

	MoveCursor(7, 23) // Start from position 4 on the next line

	fmt.Print(BgBlue + YellowHi)

	// Enable raw mode
	originalState, err := enableRawMode()
	if err != nil {
		inputLog(LogLevelError, "SysOp", "Failed to enable raw mode")
		fmt.Fprintln(os.Stderr, "Failed to enable raw mode:", err)
		os.Exit(1)
	}
	defer disableRawMode(originalState) // Restore terminal state at the end

	var r []rune
	for g.GameState.AppState != stateQuit {
		fmt.Print(BgBlue + YellowHi)
		select {
		case char := <-inputChan:
			runeChar := rune(char)
			key := string(runeChar)

			if char == '\r' || char == '\n' {
				input := sanitizeInput(strings.ToLower(string(r)))
				r = nil          // Reset buffer
				fmt.Print(Reset) // Move to the next line
				if g.GameState.AppState == stateMainMenu {
					g.handleMainMenuInput(input, inputChan, errorChan, doneChan)
				} else if g.GameState.AppState == statePlaying {
					g.handleGameplayInput(input, nil, inputChan) // Pass nil if stopChan is not needed or not available
				}
			} else if char == '\b' || char == 127 {
				if len(r) > 0 {
					r = r[:len(r)-1]   // Remove the last character from the buffer
					fmt.Print("\b \b") // Handle backspace: move cursor back, print space, move cursor back again
					g.GameState.cursX--
				}
			} else {
				// Regular character input
				fmt.Print(key) // Print character as it's typed
				r = append(r, rune(char))
				g.GameState.cursX++
				MoveCursor(g.GameState.cursX, g.GameState.cursY)
			}

		case err := <-errorChan:
			inputLog(LogLevelError, "SysOp", "Error reading input")
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
			Alias:        "SysOp",
			TimeLeft:     120 * time.Minute,
			Emulation:    1,
			NodeNum:      1,
			H:            25,
			W:            80,
			ModalH:       25,
			ModalW:       80,
			LocalDisplay: localDisplay,
		}
	} else {
		// Check for required --path argument if --local is not set
		if dropPath == "" {
			inputLog(LogLevelError, "SysOp", "Missing required -path argument")
			fmt.Fprintln(os.Stderr, "missing required -path argument")
			os.Exit(2)
		}
		user = Initialize(dropPath)
	}

	// Initialize GameState with default or initial values
	gameState := GameState{
		Door:         false,
		Pants:        true,
		Standing:     true,
		Farts:        0,
		Pills:        false,
		PillTimer:    0,
		stopTime:     false,
		cursX:        7,
		cursY:        23,
		AppState:     stateMainMenu,
		DoneChan:     make(chan bool),
		LastAppState: stateMainMenu,
		OnMainMenu:   true,
	}

	// Initialize a map for tracking awards
	awards := make(map[string]bool)

	// Initialize the Game struct with the components
	game := &Game{
		User:          user,
		GameState:     gameState,
		Awards:        awards,
		AwardedAwards: make(map[string]bool),
	}

	// Initialize User.Awards map
	game.User.Awards = make(map[string]bool)

	return game
}

func main() {
	// Open or create the log file in append mode
	file, err := os.OpenFile("game.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot open log file:", err)
	}
	defer file.Close()

	log.SetOutput(file)

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
