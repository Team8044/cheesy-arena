package field

import (
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
)

func setupAutoFuelConfig(t *testing.T) {
	previousConfig := game.ActiveGameConfig
	game.ActiveGameConfig = &game.GameConfigDefinition{
		Scoring: []game.ScoringElement{
			{Id: "fuel", Label: "Fuel", PointValue: 1, CountsForAutoFuel: true},
		},
		Panels: []game.PanelConfig{
			{
				Id: "red_near",
				Widgets: []game.WidgetConfig{
					{Id: "red_fuel", Type: "counter", ScoringId: "fuel"},
				},
			},
			{
				Id: "blue_near",
				Widgets: []game.WidgetConfig{
					{Id: "blue_fuel", Type: "counter", ScoringId: "fuel"},
				},
			},
		},
	}
	t.Cleanup(func() {
		game.ActiveGameConfig = previousConfig
	})
}

func TestResolveAutoFuelWinnerAtTeleopStart(t *testing.T) {
	arena := setupTestArena(t)
	setupAutoFuelConfig(t)

	arena.MatchState = PausePeriod
	arena.RedRealtimeScore.CurrentScore.GenericCounters = map[string]int{"red_fuel": 5}
	arena.BlueRealtimeScore.CurrentScore.GenericCounters = map[string]int{"blue_fuel": 2}

	secondsBeforeTeleop := game.MatchTiming.WarmupDurationSec + game.MatchTiming.AutoDurationSec + game.MatchTiming.PauseDurationSec - 1
	arena.MatchStartTime = time.Now().Add(-time.Duration(secondsBeforeTeleop) * time.Second)
	arena.Update()
	assert.Equal(t, PausePeriod, arena.MatchState)
	assert.Equal(t, "", arena.autoFuelWinner)
	assert.Equal(t, "", arena.currentGameSpecificMessage)

	secondsAtTeleopStart := game.MatchTiming.WarmupDurationSec + game.MatchTiming.AutoDurationSec + game.MatchTiming.PauseDurationSec
	arena.MatchStartTime = time.Now().Add(-time.Duration(secondsAtTeleopStart) * time.Second)
	arena.Update()
	assert.Equal(t, TeleopPeriod, arena.MatchState)
	assert.Equal(t, "R", arena.autoFuelWinner)
	assert.False(t, arena.autoFuelWinnerRandomized)
	assert.Equal(t, "R", arena.currentGameSpecificMessage)
}

func TestResolveAutoFuelWinnerTieRandomizesOnce(t *testing.T) {
	arena := setupTestArena(t)
	setupAutoFuelConfig(t)

	arena.RedRealtimeScore.CurrentScore.GenericCounters = map[string]int{"red_fuel": 3}
	arena.BlueRealtimeScore.CurrentScore.GenericCounters = map[string]int{"blue_fuel": 3}
	arena.randIntn = func(int) int { return 1 }

	arena.resolveAutoFuelWinner()
	assert.Equal(t, "B", arena.autoFuelWinner)
	assert.True(t, arena.autoFuelWinnerRandomized)
	assert.Equal(t, "B", arena.currentGameSpecificMessage)

	arena.randIntn = func(int) int { return 0 }
	arena.resolveAutoFuelWinner()
	assert.Equal(t, "B", arena.autoFuelWinner)
	assert.True(t, arena.autoFuelWinnerRandomized)
	assert.Equal(t, "B", arena.currentGameSpecificMessage)
}

func TestCurrentShiftStateTeleopAllianceActivity(t *testing.T) {
	arena := setupTestArena(t)
	arena.MatchState = TeleopPeriod
	arena.autoFuelWinner = "R"

	teleopStartSec := game.MatchTiming.WarmupDurationSec + game.MatchTiming.AutoDurationSec + game.MatchTiming.PauseDurationSec
	setTeleopElapsed := func(elapsedSec int) {
		arena.MatchStartTime = time.Now().Add(-time.Duration(teleopStartSec+elapsedSec) * time.Second)
	}

	setTeleopElapsed(5)
	shift := arena.currentShiftState()
	assert.Equal(t, SegmentTransition, shift.Segment)
	assert.True(t, shift.RedAllianceActive)
	assert.True(t, shift.BlueAllianceActive)

	setTeleopElapsed(15)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentShift1, shift.Segment)
	assert.False(t, shift.RedAllianceActive)
	assert.True(t, shift.BlueAllianceActive)
	assert.Equal(t, "BLUE", shift.ActiveAlliance)

	setTeleopElapsed(40)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentShift2, shift.Segment)
	assert.True(t, shift.RedAllianceActive)
	assert.False(t, shift.BlueAllianceActive)
	assert.Equal(t, "RED", shift.ActiveAlliance)

	setTeleopElapsed(65)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentShift3, shift.Segment)
	assert.False(t, shift.RedAllianceActive)
	assert.True(t, shift.BlueAllianceActive)
	assert.Equal(t, "BLUE", shift.ActiveAlliance)

	setTeleopElapsed(90)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentShift4, shift.Segment)
	assert.True(t, shift.RedAllianceActive)
	assert.False(t, shift.BlueAllianceActive)
	assert.Equal(t, "RED", shift.ActiveAlliance)

	setTeleopElapsed(120)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentEndgame, shift.Segment)
	assert.True(t, shift.RedAllianceActive)
	assert.True(t, shift.BlueAllianceActive)

	arena.autoFuelWinner = ""
	setTeleopElapsed(15)
	shift = arena.currentShiftState()
	assert.Equal(t, SegmentShift1, shift.Segment)
	assert.True(t, shift.AutoFuelWinnerPending)
	assert.Equal(t, "PENDING", shift.ActiveAlliance)
}
