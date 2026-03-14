package field

import (
	"log"

	"github.com/Team254/cheesy-arena/game"
)

type MatchSegment string

const (
	SegmentNone       MatchSegment = "none"
	SegmentAuto       MatchSegment = "auto"
	SegmentPause      MatchSegment = "pause"
	SegmentTransition MatchSegment = "transition"
	SegmentShift1     MatchSegment = "shift1"
	SegmentShift2     MatchSegment = "shift2"
	SegmentShift3     MatchSegment = "shift3"
	SegmentShift4     MatchSegment = "shift4"
	SegmentEndgame    MatchSegment = "endgame"
	SegmentPostMatch  MatchSegment = "post"
)

type ShiftState struct {
	Segment                  MatchSegment
	SegmentLabel             string
	SegmentRemainingSec      int
	AutoFuelWinner           string
	AutoFuelWinnerRandomized bool
	AutoFuelWinnerPending    bool
	ActiveAlliance           string
	RedAllianceActive        bool
	BlueAllianceActive       bool
	RedAutoFuel              int
	BlueAutoFuel             int
}

func (arena *Arena) currentShiftState() ShiftState {
	redAutoFuel, blueAutoFuel := arena.autoFuelCounts()
	shift := ShiftState{
		Segment:                  SegmentNone,
		SegmentLabel:             "",
		SegmentRemainingSec:      0,
		AutoFuelWinner:           arena.autoFuelWinner,
		AutoFuelWinnerRandomized: arena.autoFuelWinnerRandomized,
		AutoFuelWinnerPending:    false,
		ActiveAlliance:           "NONE",
		RedAllianceActive:        false,
		BlueAllianceActive:       false,
		RedAutoFuel:              redAutoFuel,
		BlueAutoFuel:             blueAutoFuel,
	}

	matchTimeSec := int(arena.MatchTimeSec())
	switch arena.MatchState {
	case AutoPeriod:
		shift.Segment = SegmentAuto
		shift.SegmentLabel = "AUTO"
		shift.SegmentRemainingSec = max(0, game.MatchTiming.WarmupDurationSec+game.MatchTiming.AutoDurationSec-matchTimeSec)
		shift.ActiveAlliance = "BOTH"
		shift.RedAllianceActive = true
		shift.BlueAllianceActive = true
	case PausePeriod:
		shift.Segment = SegmentPause
		shift.SegmentLabel = "PENDING"
		shift.SegmentRemainingSec = max(
			0, game.MatchTiming.WarmupDurationSec+game.MatchTiming.AutoDurationSec+game.MatchTiming.PauseDurationSec-matchTimeSec,
		)
		shift.AutoFuelWinnerPending = true
		shift.ActiveAlliance = "PENDING"
	case TeleopPeriod:
		teleopStartSec := game.MatchTiming.WarmupDurationSec + game.MatchTiming.AutoDurationSec + game.MatchTiming.PauseDurationSec
		teleopElapsed := max(0, matchTimeSec-teleopStartSec)
		transitionEnd := 10
		shift1End := transitionEnd + 25
		shift2End := shift1End + 25
		shift3End := shift2End + 25
		shift4End := shift3End + 25
		endgameEnd := shift4End + 30

		switch {
		case teleopElapsed < transitionEnd:
			shift.Segment = SegmentTransition
			shift.SegmentLabel = "TRANSITION"
			shift.SegmentRemainingSec = transitionEnd - teleopElapsed
			shift.ActiveAlliance = "BOTH"
			shift.RedAllianceActive = true
			shift.BlueAllianceActive = true
		case teleopElapsed < shift1End:
			shift.Segment = SegmentShift1
			shift.SegmentLabel = "SHIFT 1"
			shift.SegmentRemainingSec = shift1End - teleopElapsed
			shift.RedAllianceActive, shift.BlueAllianceActive, shift.ActiveAlliance = getAllianceShiftActivity(
				arena.autoFuelWinner, true,
			)
		case teleopElapsed < shift2End:
			shift.Segment = SegmentShift2
			shift.SegmentLabel = "SHIFT 2"
			shift.SegmentRemainingSec = shift2End - teleopElapsed
			shift.RedAllianceActive, shift.BlueAllianceActive, shift.ActiveAlliance = getAllianceShiftActivity(
				arena.autoFuelWinner, false,
			)
		case teleopElapsed < shift3End:
			shift.Segment = SegmentShift3
			shift.SegmentLabel = "SHIFT 3"
			shift.SegmentRemainingSec = shift3End - teleopElapsed
			shift.RedAllianceActive, shift.BlueAllianceActive, shift.ActiveAlliance = getAllianceShiftActivity(
				arena.autoFuelWinner, true,
			)
		case teleopElapsed < shift4End:
			shift.Segment = SegmentShift4
			shift.SegmentLabel = "SHIFT 4"
			shift.SegmentRemainingSec = shift4End - teleopElapsed
			shift.RedAllianceActive, shift.BlueAllianceActive, shift.ActiveAlliance = getAllianceShiftActivity(
				arena.autoFuelWinner, false,
			)
		case teleopElapsed < endgameEnd:
			shift.Segment = SegmentEndgame
			shift.SegmentLabel = "ENDGAME"
			shift.SegmentRemainingSec = endgameEnd - teleopElapsed
			shift.ActiveAlliance = "BOTH"
			shift.RedAllianceActive = true
			shift.BlueAllianceActive = true
		default:
			shift.Segment = SegmentPostMatch
			shift.SegmentLabel = "POST-MATCH"
		}

		if shift.ActiveAlliance == "PENDING" {
			shift.AutoFuelWinnerPending = true
		}
	case PostMatch:
		shift.Segment = SegmentPostMatch
		shift.SegmentLabel = "POST-MATCH"
	}

	return shift
}

func getAllianceShiftActivity(autoFuelWinner string, loserShift bool) (bool, bool, string) {
	if autoFuelWinner == "" {
		return false, false, "PENDING"
	}
	redActive := autoFuelWinner == "B"
	blueActive := autoFuelWinner == "R"
	if !loserShift {
		redActive = !redActive
		blueActive = !blueActive
	}
	if redActive {
		return true, false, "RED"
	}
	if blueActive {
		return false, true, "BLUE"
	}
	return false, false, "NONE"
}

func (arena *Arena) autoFuelCounts() (int, int) {
	redAutoFuel := arena.RedRealtimeScore.CurrentScore.AutoFuelCountFromConfig()
	blueAutoFuel := arena.BlueRealtimeScore.CurrentScore.AutoFuelCountFromConfig()
	return redAutoFuel, blueAutoFuel
}

func (arena *Arena) resolveAutoFuelWinner() {
	if arena.autoFuelWinner != "" {
		return
	}

	redAutoFuel, blueAutoFuel := arena.autoFuelCounts()
	winner := ""
	randomized := false
	if redAutoFuel > blueAutoFuel {
		winner = "R"
	} else if blueAutoFuel > redAutoFuel {
		winner = "B"
	} else {
		randomized = true
		if arena.randIntn(2) == 0 {
			winner = "R"
		} else {
			winner = "B"
		}
	}

	arena.autoFuelWinner = winner
	arena.autoFuelWinnerRandomized = randomized
	arena.setGameSpecificMessage(winner)

	if randomized {
		log.Printf(
			"AUTO fuel tied at R=%d, B=%d; randomly selected %s for SHIFT 2/4 and game data", redAutoFuel, blueAutoFuel,
			winner,
		)
	} else {
		log.Printf("AUTO fuel winner %s at R=%d, B=%d", winner, redAutoFuel, blueAutoFuel)
	}
}

func (arena *Arena) setGameSpecificMessage(message string) {
	arena.currentGameSpecificMessage = message
}
