package main

import (
	"fmt"
	"strings"
)

type Award struct {
	ID              string
	Name            string
	Description     string
	Art             string
	AwardConditions []string
	OnMainMenu      bool
}

var awards = []Award{
	{
		ID:              "award1",
		Name:            "Thinking (and shitting) inside the box",
		Description:     "Congratulations, all that potty training finally paid off.",
		Art:             "award3.ans",
		AwardConditions: []string{"fart lightly", "pull door", "remove pants", "go to bathroom", "shit"},
		OnMainMenu:      false,
	},
	{
		ID:              "award3",
		Name:            "Shitting 101",
		Description:     "Sometimes even zero effort is rewarded.",
		Art:             "award3.ans",
		AwardConditions: []string{"shit"},
		OnMainMenu:      false,
	},
	{
		ID:              "award8",
		Name:            "Shitting at the starting gun",
		Description:     "You shit before the game began!",
		Art:             "award8.ans",
		AwardConditions: []string{"shit"},
		OnMainMenu:      true,
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
