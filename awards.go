package main

import (
	"fmt"
	"strings"
)

type Award struct {
	ID              string
	Name            string
	Description     string
	AwardConditions []string // conditions to earn the award
	RunDownClock    bool     // award is earned by letting the clock run down
	OnMainMenu      bool     // award is earned from the main menu
	Required        []string // required awards to earn this one
	Optional        []string // optional awards to earn this one
}

var awards = []Award{
	{
		ID:              "1",
		Name:            "Thinking (and shitting) inside the box",
		Description:     "Congratulations, all that potty training finally paid off.",
		AwardConditions: []string{"pull door", "remove pants", "sit toilet", "shit"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "2",
		Name:            "Mr. Efficient",
		Description:     "It's not his fault that door was so hard to open.",
		AwardConditions: []string{"remove pants", "shit"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "3",
		Name:            "Shitting 101",
		Description:     "Sometimes even zero effort is rewarded.", // typed shit, or farted, from game input
		AwardConditions: []string{"shit"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "4",
		Name:            "So close and yet so far...",
		Description:     "Pants. They get you every time.",
		AwardConditions: []string{"pull door", "sit toilet", "shit"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "5",
		Name:            "Sep-poo-ku",
		Description:     "Giving up is never the answer. Or is it?",
		AwardConditions: []string{"suicide"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        []string{"remove pants"},
	},
	{
		ID:              "6",
		Name:            "Holding off the inevitable",
		Description:     "How convenient that you had those pills...",
		AwardConditions: []string{"take pills", "fart gently"}, // let time run out
		RunDownClock:    true,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "7",
		Name:            "The inevitable...",
		Description:     "...to not to poop for an extra five seconds", // granted immediately after award 6
		AwardConditions: []string{"take pills", "fart gently"},
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        []string{"6"},
		Optional:        nil,
	},
	{
		ID:              "8",
		Name:            "Shitting at the starting gun",
		Description:     "You shit before the game began!",
		AwardConditions: []string{"shit"}, // typed "shit" from the main menu
		RunDownClock:    false,
		OnMainMenu:      true,
		Required:        nil,
		Optional:        nil,
	},
	{
		ID:              "9",
		Name:            "Slow typer",
		Description:     "If only you had a little more time... and a higher IQ.",
		AwardConditions: []string{""},
		RunDownClock:    true,
		OnMainMenu:      false,
		Required:        nil,
		Optional:        []string{"remove pants"},
	},
	{
		ID:              "10",
		Name:            "Final Award: You are the Shit King!",
		Description:     "And you have a crown to prove it!",
		AwardConditions: []string{"shit"}, // Earn the first 9 awards. Automatic
		RunDownClock:    false,
		OnMainMenu:      false,
		Required:        []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
	},

	// Define more awards as needed
}

func (g *Game) checkAndGrantAwards(inputChan chan byte) {
	// Check if the user is in the main menu state
	isMainMenu := g.GameState.AppState == stateMainMenu

	// Define a map to associate specific input words with their corresponding "main" words
	var wordMappings = map[string]string{
		"poop": "shit",
		// Add more mappings as needed
	}

	// Iterate over awards
	for _, award := range awards {
		// Check if the user has not already earned the award and is eligible to earn it based on MainMenu field
		if !g.User.Awards[award.ID] && ((award.OnMainMenu && isMainMenu) || (!award.OnMainMenu && !isMainMenu)) {
			// Initialize a flag to track if all conditions are met
			allConditionsMet := true

			// Iterate over conditions for the award
			for _, condition := range award.AwardConditions {
				// Split the condition into words
				conditionWords := strings.Fields(condition)

				// Initialize a flag to track if any condition word is met
				conditionMet := false

				// Iterate over input buffer
				for _, input := range g.UserInputBuffer {
					// Iterate over condition words
					for _, conditionWord := range conditionWords {
						conditionWord = strings.ToLower(conditionWord)

						// Check if any input word matches any condition word from any list
						if containsWordFromLists(input, conditionWord, verbList) {
							conditionMet = true
							break
						}
					}

					// Check if the input word matches a mapped "main" word
					if mainWord, ok := wordMappings[input]; ok {
						for _, conditionWord := range conditionWords {
							conditionWord = strings.ToLower(conditionWord)

							// Check if the mapped "main" word matches the condition word from any list
							if containsWordFromLists(mainWord, conditionWord, verbList) {
								conditionMet = true
								break
							}
						}
					}

					if conditionMet {
						break
					}
				}

				// If any condition word is not met, set allConditionsMet to false
				if !conditionMet {
					allConditionsMet = false
					break
				}
			}

			// If all conditions are met, grant the award and mark it as earned
			if allConditionsMet {
				g.User.Awards[award.ID] = true

				// Display the award message
				ClearScreen()
				fmt.Printf("Congratulations! You've earned the %s award!\n", award.Name)

				// Pause for a keypress
				g.readSingleKeyPress(inputChan, stateAwards)

				// Clear the input buffer here
				g.UserInputBuffer = []string{}

				// Exit the loop after granting an award
				return
			}
		}
	}
}

// Function to check if a word exists in any of the specified lists
func containsWordFromLists(word, conditionWord string, lists map[string][]string) bool {
	for _, list := range lists {
		for _, item := range list {
			if strings.Contains(word, item) && strings.Contains(conditionWord, item) {
				return true
			}
		}
	}
	return false
}

// Function to get the award name by ID
func getAwardNameByID(awardID string) string {
	for _, award := range awards {
		if award.ID == awardID {
			return award.Name
		}
	}
	return "Unknown Award"
}
