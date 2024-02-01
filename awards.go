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
}

var awards = []Award{
	{
		ID:              "award1",
		Name:            "Thinking (and shitting) inside the box",
		Description:     "Congratulations, all that potty training finally paid off.",
		Art:             "award3.ans",
		AwardConditions: []string{"play", "fart lightly", "pull door", "remove pants", "go to bathroom", "shit"},
	},
	{
		ID:              "award3",
		Name:            "Shitting 101",
		Description:     "Sometimes even zero effort is rewarded.",
		Art:             "award3.ans",
		AwardConditions: []string{"play", "shit"},
	},
	{
		ID:              "award8",
		Name:            "Shitting at the starting gun",
		Description:     "You shit before the game began!",
		Art:             "award8.ans",
		AwardConditions: []string{"shit"},
	},

	// Define more awards as needed
}

func matchesCondition(inputBuffer []string, conditions []string) bool {
	for _, condition := range conditions {
		condition = strings.TrimSpace(strings.ToLower(condition))
		found := false

		// Check if the condition exists in the inputBuffer
		for _, input := range inputBuffer {
			input = strings.TrimSpace(strings.ToLower(input))
			if input == condition {
				found = true
				break
			}
		}

		// If the condition is not found in the inputBuffer, return false
		if !found {
			return false
		}
	}

	// If all conditions are found, return true
	return true
}

func (g *Game) checkAndGrantAwards(inputChan chan byte) {
	// Check if the user has earned any awards
	for _, award := range awards {
		if matchesCondition(g.UserInputBuffer, award.AwardConditions) {
			if award.ID == "award8" {

				// Grant the award to the user
				g.User.Awards[award.ID] = true
				// Display the award art here
				//awardArt := award.Art
				//displayAnsiFile(ArtFileDir+awardArt, g.User.LocalDisplay)
				ClearScreen()
				fmt.Print("Congratulations! You've earned the Shitting at the starting gun award!\n")

				// Pause for a keypress
				g.readSingleKeyPress(inputChan, stateAwards)

				// Clear the input buffer here
				g.UserInputBuffer = []string{}

				// Break out of the loop after granting an award
				return

			} else {
				// For other awards, grant them normally
				g.User.Awards[award.ID] = true
				ClearScreen()
				// You can also display the award art here
				fmt.Printf("Congratulations! You've earned the %s award!\n", award.Name)

				// Pause for a keypress
				g.readSingleKeyPress(inputChan, stateAwards)

				// Clear the input buffer here
				g.UserInputBuffer = []string{}

				// Break out of the loop after granting an award
				return
			}
		}
	}
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
