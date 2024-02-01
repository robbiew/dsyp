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
	Sequence        []string
	AwardConditions []string
}

var awards = []Award{
	{
		ID:              "award3",
		Name:            "Shitting 101",
		Description:     "Sometimes even zero effort is rewarded.",
		Art:             "award3.ans",
		Sequence:        []string{"shit"},
		AwardConditions: []string{"shit"},
	},

	// Define more awards as needed
}

func matchesCondition(inputBuffer []string, conditions []string) bool {
	if len(inputBuffer) < len(conditions) {
		return false
	}

	for i, condition := range conditions {
		input := strings.TrimSpace(strings.ToLower(inputBuffer[i]))
		condition = strings.TrimSpace(strings.ToLower(condition))
		if input != condition {
			return false
		}
	}
	return true
}

func (g *Game) checkAndGrantAwards() {
	// Check if the user has earned any awards
	for _, award := range awards {
		if matchesCondition(g.UserInputBuffer, award.AwardConditions) {
			// Grant the award to the user
			g.User.Awards[award.ID] = true
			// You can also display the award art here
			fmt.Printf("Congratulations! You've earned the %s award!\n", award.Name)
		}
	}
}
