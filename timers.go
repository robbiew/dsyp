package main

import (
	"fmt"
	"time"
)

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
			if remaining < time.Second*20 {
				// Specific logic when the timer is under 20 seconds
				MoveCursor(2, 23)
				fmt.Print(EraseLine)
				fmt.Println(BgBlue + RedHi + "Hurry! You need to find a way to reduce the pressure in your gut." + Reset)
			}

			// Timer update logic
			MoveCursor(0, 0)
			fmt.Print(Reset + EraseLine)
			fmt.Printf(Reset+Green+" TIMER: %v"+Reset, remaining)
			fmt.Print(BgBlue + YellowHi)

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
