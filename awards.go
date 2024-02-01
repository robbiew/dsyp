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

	// Create a map to store verb and noun lists
	lists := map[string][]string{
		"poopVerbs":      poopVerbs,
		"lookVerbs":      lookVerbs,
		"openVerbs":      openVerbs,
		"breakVerbs":     breakVerbs,
		"pullVerbs":      pullVerbs,
		"closeVerbs":     closeVerbs,
		"removeVerbs":    removeVerbs,
		"wearVerbs":      wearVerbs,
		"moveVerbs":      moveVerbs,
		"eatVerbs":       eatVerbs,
		"dieVerbs":       dieVerbs,
		"lightlyAdverbs": lightlyAdverbs,
		"bathroomNouns":  bathroomNouns,
		"pillsNouns":     pillsNouns,
	}

	// Iterate over awards
	for _, award := range awards {
		// Check if the award is eligible to be granted based on the MainMenu field
		if (award.OnMainMenu && g.GameState.OnMainMenu) || (!award.OnMainMenu && !g.GameState.OnMainMenu) {
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
					// Check if any input word matches any condition word from any list
					for _, conditionWord := range conditionWords {
						conditionWord = strings.ToLower(conditionWord)
						if containsWordFromLists(input, conditionWord, lists) {
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
		} else {
			// no award to grant
			ClearScreen()

			// Clear the input buffer here
			g.UserInputBuffer = []string{}
			return
		}
	}
}

// Function to check if a word contains any of the possible verb or noun options
func containsWordFromLists(word, conditionWord string, lists map[string][]string) bool {
	conditionWord = strings.ToLower(conditionWord)
	for _, list := range lists {
		for _, option := range list {
			option = strings.ToLower(option)
			if strings.Contains(word, option) && strings.Contains(conditionWord, option) {
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
