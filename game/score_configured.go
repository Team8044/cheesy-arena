package game

// summarizeFromConfig produces a score summary based on the active configurable game settings.
func (score *Score) summarizeFromConfig(opponentScore *Score) *ScoreSummary {
	summary := new(ScoreSummary)
	if score.PlayoffDq {
		return summary
	}

	// Derive scoring counts on the fly instead of persisting to avoid stale accumulation.
	scoringCounts := score.configuredScoringCounts()
	summary.MatchPoints += score.pointsFromUnscoredConfiguredWidgets()

	// Apply scoring element point values.
	for _, scoring := range ActiveGameConfig.Scoring {
		count := scoringCounts[scoring.Id]
		summary.MatchPoints += count * scoring.PointValue
	}

	// Fouls assessed by the opponent.
	for _, foul := range opponentScore.Fouls {
		summary.FoulPoints += foul.PointValue()
		if foul.IsMajor {
			summary.NumOpponentMajorFouls++
		}
	}

	summary.Score = summary.MatchPoints + summary.FoulPoints
	return summary
}

// AutoFuelCountFromConfig computes the number of configured scoring elements marked as AUTO fuel.
func (score *Score) AutoFuelCountFromConfig() int {
	if ActiveGameConfig == nil {
		return 0
	}
	scoringCounts := score.configuredScoringCounts()
	autoFuelCount := 0
	for _, scoring := range ActiveGameConfig.Scoring {
		if scoring.CountsForAutoFuel {
			autoFuelCount += scoringCounts[scoring.Id]
		}
	}
	return autoFuelCount
}

func (score *Score) configuredScoringCounts() map[string]int {
	scoringCounts := map[string]int{}
	if ActiveGameConfig == nil {
		return scoringCounts
	}

	for widgetId, value := range score.GenericCounters {
		if widget := ActiveGameConfig.WidgetById(widgetId); widget != nil && widget.ScoringId != "" {
			scoringCounts[widget.ScoringId] += value
		}
	}

	for widgetId, value := range score.GenericToggles {
		if !value {
			continue
		}
		if widget := ActiveGameConfig.WidgetById(widgetId); widget != nil && widget.ScoringId != "" {
			scoringCounts[widget.ScoringId]++
		}
	}

	for widgetId, state := range score.GenericStates {
		if state == "" {
			continue
		}
		widget := ActiveGameConfig.WidgetById(widgetId)
		if widget == nil {
			continue
		}
		if widget.Type == "multistate" {
			for _, st := range widget.States {
				if st.Value == state && st.ScoringId != "" {
					scoringCounts[st.ScoringId]++
					break
				}
			}
			continue
		}
		if _, ok := widget.StatePoints[state]; ok && widget.ScoringId != "" {
			scoringCounts[widget.ScoringId]++
		}
	}

	return scoringCounts
}

func (score *Score) pointsFromUnscoredConfiguredWidgets() int {
	if ActiveGameConfig == nil {
		return 0
	}
	points := 0

	for widgetId, value := range score.GenericCounters {
		if widget := ActiveGameConfig.WidgetById(widgetId); widget != nil && widget.ScoringId == "" {
			points += value * widget.PointValue
		}
	}

	for widgetId, value := range score.GenericToggles {
		if !value {
			continue
		}
		if widget := ActiveGameConfig.WidgetById(widgetId); widget != nil && widget.ScoringId == "" {
			points += widget.PointValue
		}
	}

	for widgetId, state := range score.GenericStates {
		if state == "" {
			continue
		}
		widget := ActiveGameConfig.WidgetById(widgetId)
		if widget == nil || widget.Type == "multistate" {
			continue
		}
		if statePoints, ok := widget.StatePoints[state]; ok {
			points += statePoints
		}
	}

	return points
}
