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
	MainMenu        bool
}

var awards = []Award{
	{
		ID:              "award1",
		Name:            "Thinking (and shitting) inside the box",
		Description:     "Congratulations, all that potty training finally paid off.",
		Art:             "award3.ans",
		AwardConditions: []string{"fart lightly", "pull door", "remove pants", "go to bathroom", "shit"},
		MainMenu:        false,
	},
	{
		ID:              "award3",
		Name:            "Shitting 101",
		Description:     "Sometimes even zero effort is rewarded.",
		Art:             "award3.ans",
		AwardConditions: []string{"shit"},
		MainMenu:        false,
	},
	{
		ID:              "award8",
		Name:            "Shitting at the starting gun",
		Description:     "You shit before the game began!",
		Art:             "award8.ans",
		AwardConditions: []string{"shit"},
		MainMenu:        true,
	},

	// Define more awards as needed
}

func (g *Game) checkAndGrantAwards(inputChan chan byte) {
	// Check if the user is in the main menu state
	isMainMenu := g.GameState.AppState == stateMainMenu

	// Iterate over awards
	for _, award := range awards {
		// Check if the award is eligible to be granted based on the MainMenu field
		if (award.MainMenu && isMainMenu) || (!award.MainMenu && !isMainMenu) {
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
					// Check if any input word matches any condition word
					for _, conditionWord := range conditionWords {
						if isPoopVerb(input) && containsWord(input, conditionWord) {
							conditionMet = true
							break
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

			// If all conditions are met, grant the award
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

func isPoopVerb(word string) bool {
	poopVerbs := []string{"poop", "poo", "crap", "dump", "shit", "defecate"} // Add your poop verbs here
	word = strings.TrimSpace(strings.ToLower(word))
	for _, verb := range poopVerbs {
		if verb == word {
			return true
		}
	}
	return false
}

func containsWord(input, condition string) bool {
	return strings.Contains(input, condition)
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
