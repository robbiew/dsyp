package main

import (
	"fmt"
	"time"
)

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
