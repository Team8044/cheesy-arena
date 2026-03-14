# 2026 Implementation Plan

Status: implemented (software baseline complete; sensor integration pending)  
Owner: Team 8044  
Purpose: track the work needed to adapt Cheesy Arena to the 2026 FRC game

## Scope

Required changes from the current request:

1. Change the default autonomous period from 15 seconds to 20 seconds.
2. Implement 2026 teleop shift logic:
   - Transition Shift: 10 seconds
   - Shift 1: 25 seconds
   - Shift 2: 25 seconds
   - Shift 3: 25 seconds
   - Shift 4: 25 seconds
   - End Game: 30 seconds
3. Restrict alliance scoring so an alliance may only score during:
   - its active alliance shifts,
   - the Transition Shift,
   - the End Game.
4. Determine which alliance is active in Shift 1/3 vs Shift 2/4 from AUTO fuel scoring on the manual scoring tablets.
5. Send the official single-character game-specific message (`"R"` or `"B"`) to driver stations.
6. Show the current shift and active alliance information on every relevant operator and audience surface.

## Implementation Status (March 14, 2026)

Completed:

- Default timing updated to `AUTO=20s`, `PAUSE=3s`, `TELEOP=140s`.
- 2026 shift segmentation implemented and published in match-time websocket messages:
  - Transition, Shift 1-4, Endgame.
- AUTO fuel winner determination implemented from manual scoring panel data (config-tagged scoring elements).
- Winner is finalized exactly at teleop start (after pause), with tie randomization (`R`/`B`) matching official behavior.
- Driver Station game-specific message (`"R"` / `"B"`) is sent on the existing DS TCP game-data path.
- Shift state and active-alliance indicators are visible across relevant web UIs:
  - audience, wall, announcer, field monitor, alliance station, referee panel, match play, scoring panels.
- Game Config Builder supports:
  - per-scoring-element `AUTO Fuel` tagging,
  - expanded panel phase options (`transition`, `shift1..4`, `teleop_any`, etc.).
- Scoring panel behavior remains advisory (no hard disable), with explicit active/inactive shift status shown to scorers.

Pending / later follow-up:

- Automatic sensor-based fuel scoring integration (current source remains manual scoring panels).

## Official Rule / Interface Notes

Summary of the official behavior:

- AUTO is 20 seconds.
- There is a 3-second scoring delay between AUTO and TELEOP.
- TELEOP is 2:20 total and is segmented into Transition, Shift 1, Shift 2, Shift 3, Shift 4, and End Game.
- The alliance that scored more fuel during AUTO is active in Shift 2 and Shift 4, and inactive in Shift 1 and Shift 3.
- If AUTO fuel is tied, official FMS selects an alliance for that role. Cheesy Arena should mirror that behavior.
- The game-specific message is the single character `R` or `B`.
- Official docs say the game-specific message is transmitted at the start of TELEOP / approximately 3 seconds after AUTO ends, not immediately at the instant AUTO expires. Cheesy Arena should mirror that timing.

Important inference:

- I found official documentation for the meaning and timing of the game-specific message, but not a public FIRST packet-level protocol spec for the DS TCP frame itself.
- Cheesy Arena already has `field/driver_station_connection.go` support for sending game data over the DS TCP channel via `sendGameDataPacket()`.
- Plan assumption: reuse that existing transport path and validate it with local tests, plus packet capture or a real DS sanity check if available.

## Current Codebase Findings

### Timing and match state

- `game/match_timing.go` still defaults to `AutoDurationSec=15`, `PauseDurationSec=3`, `TeleopDurationSec=135`.
- `field/arena.go` only models coarse states:
  - `PreMatch`, `WarmupPeriod`, `AutoPeriod`, `PausePeriod`, `TeleopPeriod`, `PostMatch`, timeout states.
- `static/js/match_timing.js` only translates those coarse states. There is no concept of Transition Shift, Shift 1-4, or active alliance windows.

### Scoring

- The dynamic scoring system is now driven by the configurable game config in `game/config.go`, `game/score_configured.go`, `web/scoring_panel.go`, and `static/js/scoring_panel.js`.
- Widget phase gating currently supports only:
  - `any`
  - `auto`
  - `teleop`
  - `endgame`
- There is no backend concept of “this alliance is inactive right now,” so the scoring tablets cannot currently enforce shift legality.
- The configurable score summarizer does not expose per-scoring-element counts outside the summary routine, which means AUTO fuel winner selection logic does not yet have a clean reusable input.

### Displays and operator surfaces

- Relevant display surfaces already subscribe to websocket updates and can be extended:
  - audience: `web/audience_display.go`, `static/js/audience_display.js`
  - wall: `web/wall_display.go`, `static/js/wall_display.js`
  - alliance station display: `web/alliance_station_display.go`, `static/js/alliance_station_display.js`
  - announcer display: `web/announcer_display.go`, `static/js/announcer_display.js`
  - field monitor: `web/field_monitor_display.go`, `static/js/field_monitor_display.js`
  - match play / admin: `web/match_play.go`, `static/js/match_play.js`
  - referee panel: `web/referee_panel.go`, `static/js/referee_panel.js`

### Driver station communication

- `field/driver_station_connection.go` already has:
  - control packet encode/send logic,
  - team assignment TCP handshake,
  - `sendGameDataPacket(gameData string)`.
- There is currently no call site that actually sends a game-specific message during a match.

## Proposed Architecture

## 1. Match timing and shift model

Add a new backend concept that sits on top of the coarse match state:

- `MatchSegment` enum or equivalent:
  - `SegmentNone`
  - `SegmentAuto`
  - `SegmentTransition`
  - `SegmentShift1`
  - `SegmentShift2`
  - `SegmentShift3`
  - `SegmentShift4`
  - `SegmentEndgame`
  - `SegmentPostMatch`
- `ShiftState` or equivalent helper payload:
  - current segment
  - segment label
  - countdown within the segment
  - total match countdown
  - active scoring alliances
  - AUTO fuel winner / shift-seed alliance
  - game-specific message character, if known

Recommended implementation points:

- `game/match_timing.go`
- `field/arena.go`
- `field/arena_notifiers.go`
- `static/js/match_timing.js`

Notes:

- Default timing should become `Auto=20`, `Pause=3`, `Teleop=140`.
- Existing coarse `AutoPeriod`, `PausePeriod`, `TeleopPeriod` can stay, but shift logic should be derived from elapsed time.
- End Game should remain inside `TeleopPeriod`; it does not need a new top-level `MatchState`.

## 2. AUTO fuel winner and shift-order determination

We need a single source of truth for “which alliance is the Shift 2/4 alliance.”

Recommended model:

- Track derived AUTO fuel counts from scoring input only.
- There should be no manual override path.
- Fuel entered during the 3-second pause still counts toward AUTO winner determination.
- The final shift seed alliance should be determined at the exact end of the 3-second pause.
- If the counts are tied at that instant, Cheesy Arena should randomly select `R` or `B`, mirroring official FMS behavior.

Why this is needed:

- Manual scoring may still be completed during the 3-second pause.
- Tie handling requires an explicit random-selection path.
- The game-specific message needs a stable cached value once the selection is finalized at TELEOP start.

Implementation options:

- Preferred: add dedicated live match-level shift fields, not hidden state in generic widget maps.
- Candidate storage locations:
  - `field.Arena` for live state,
  - `model.MatchResult` only if review persistence becomes useful later,
  - possibly `game.Score` only for raw per-alliance counts, not for the final cross-alliance decision.

Recommended derived data helper:

- Add a reusable function that expands configurable widget state into scoring-element counts by `ScoringId`.
- Use configuration to identify which `ScoringId` values count as AUTO fuel for shift determination.

This likely implies extending the game configuration schema with something like:

- `shiftDetermination.autoFuelScoringIds`
- `shiftDetermination.tieBehavior`

Current direction:

- Include AUTO fuel configuration in the game config builder rather than hardcoding it.
- The manually operated scoring panels are the authoritative fuel source until an automatic scoring system exists.
- The game config should explicitly tag which scoring elements count toward AUTO fuel winner determination.

Files likely involved:

- `game/config.go`
- `model/game_config.go`
- `game/score_configured.go`
- `web/setup_game_config.go`
- `templates/setup_game_config.html`
- `static/js/game_config_builder.js`
- `model/match_result.go`

## 3. Scoring legality enforcement on manual tablets

The tablets need to stop or clearly block illegal scoring during inactive alliance shifts.

Recommended behavior:

- During Transition Shift and End Game:
  - both alliances may score.
- During Shift 1 and Shift 3:
  - only the alliance designated for Shift 1/3 may score.
- During Shift 2 and Shift 4:
  - only the alliance designated for Shift 2/4 may score.

UI behavior recommendation:

- Do not hard-disable scoring widgets on inactive-alliance panels.
- Show a prominent banner:
  - current shift name,
  - active alliance,
  - whether the local panel alliance is active or inactive.
- Keep foul entry available regardless of shift.
- Keep post-match commit workflow unchanged.

Operational note:

- The panel UI is advisory for the scorer.
- The backend should not reject scoring events based on active/inactive shift status.
- Cheesy Arena should expose the active/inactive state clearly and trust the scorer.

Game-config changes needed:

- Replace the current coarse widget `phase` model with a richer set:
  - `any`
  - `auto`
  - `transition`
  - `shift1`
  - `shift2`
  - `shift3`
  - `shift4`
  - `endgame`
  - optionally `teleop_any`
- Update builder UI and runtime gating logic accordingly.

Files likely involved:

- `templates/scoring_panel.html`
- `static/js/scoring_panel.js`
- `web/scoring_panel.go`
- `templates/setup_game_config.html`
- `static/js/game_config_builder.js`
- `game/config.go`

## 4. Display and operator-surface updates

Expose the new shift state everywhere an operator or audience needs it.

Surfaces to update:

- audience display
- wall display
- announcer display
- alliance station display
- field monitor display
- match play / admin page
- referee panel
- scoring tablets

Recommended shared data payload:

- extend `MatchTimeMessage` or add a dedicated `MatchShiftMessage` with:
  - current segment id
  - current segment label
  - segment countdown
  - active alliances
  - shift-seed alliance
  - game-specific message char

Display expectations:

- Audience / wall / announcer:
  - show the shift name and active alliance prominently near the match timer.
- Alliance station display:
  - show a compact “ACTIVE”, “INACTIVE”, or “BOTH ACTIVE” state for the local alliance.
- Field monitor / match play / referee:
  - show current shift and shift-seed alliance for field staff.

Files likely involved:

- `field/arena_notifiers.go`
- `static/js/match_timing.js`
- `static/js/audience_display.js`
- `static/js/wall_display.js`
- `static/js/announcer_display.js`
- `static/js/alliance_station_display.js`
- `static/js/field_monitor_display.js`
- `static/js/match_play.js`
- `static/js/referee_panel.js`

## 5. Driver station game-specific message

Required runtime behavior:

- Before the AUTO winner is resolved:
  - send no decision / empty string.
- When the final shift-seed alliance is known:
  - send `"R"` or `"B"` to all attached driver stations.
- Cache the resolved value for the current match.
- Re-send the cached value to any DS that connects or reconnects after the decision is known.

Recommended timing behavior:

- Match official docs as closely as possible:
  - finalize and send at the start of TELEOP / after the 3-second AUTO scoring delay.
- No early-release path.

Implementation points:

- add live cached field on `Arena` for current match game data
- set/clear it when matches load, start, abort, or are discarded
- trigger DS send when:
  - the game data changes,
  - a DS connects after the decision exists

Files likely involved:

- `field/arena.go`
- `field/driver_station_connection.go`
- `field/driver_station_connection_test.go`

## 6. Persistence and replay behavior

Recommended persistence:

- Persistence is low priority for this scrimmage use case.
- Live operation is the priority.
- Optional review persistence can be added later if needed.
- Runtime-only state is acceptable for the first implementation as long as live displays and driver stations behave correctly.

Minimum persistent fields recommended:

- derived red AUTO fuel count
- derived blue AUTO fuel count
- final shift-seed alliance (`R` or `B`)
- whether the selection came from a tie-randomized result

Files likely involved:

- `model/match_result.go`
- `web/match_review.go`
- tests around match review and result serialization

## 7. Testing plan

### Backend unit tests

- `game/match_timing.go`
  - default durations become 20 / 3 / 140.
- `field/arena.go`
  - segment boundaries at exact second transitions:
    - AUTO -> Transition
    - Transition -> Shift 1
    - Shift 1 -> Shift 2
    - Shift 2 -> Shift 3
    - Shift 3 -> Shift 4
    - Shift 4 -> End Game
    - End Game -> Post Match
- shift active-alliance logic for both `R` and `B` seed cases
- tie / unresolved cases
- game-data caching and send timing

### Driver station tests

- `field/driver_station_connection_test.go`
  - send `R` / `B` over existing TCP game-data path
  - resend on late DS connection
  - clear between matches

### Websocket / UI tests

- extend websocket message tests for new shift metadata
- scoring panel tests:
  - active/inactive panel state updates correctly at segment boundaries
  - current shift and active-alliance indicators stay synchronized
- audience / alliance station / field monitor / match play tests for new text fields if those tests exist today

### Manual validation

- dry run an entire match with both possible AUTO winners
- verify tablet gating changes exactly at segment boundaries
- verify audience and team displays stay synchronized
- verify DS receives the correct game-specific message on a real driver station if available

## Recommended Implementation Order

1. Timing defaults and backend shift-state model.
2. AUTO fuel count derivation and tie-randomization model.
3. Websocket payload changes.
4. Scoring tablet shift-awareness UI.
5. Driver station game-data transmission.
6. Audience/station/operator display updates.
7. Optional persistence, review surfaces, and regression cleanup.
8. Full test pass and live dry run.

## Risks

- The existing configurable scoring model does not yet identify “AUTO fuel” cleanly; that needs explicit metadata in the game config builder.
- Random tie-breaking should be deterministic enough to test, but still behave as random in production.
- The winner is intentionally unresolved until the exact end of the pause, so UI state during the pause must stay clear.
- Because scoring remains advisory during inactive shifts, scorer discipline matters; the system will surface state but not block input.

## Open Questions For Signoff

No outstanding signoff decisions are currently required for the planned first implementation.

Resolved implementation decisions:

1. Backend scoring enforcement is advisory-only; it should not reject inactive-shift scoring input.
2. The game config builder should explicitly mark which scoring elements count toward AUTO fuel winner determination.
3. Pseudorandom tie resolution at TELEOP start is acceptable, and the selected alliance should be visible or logged.
4. During the 3-second pause, displays should show the shift seed as `PENDING`.
5. Runtime-only state is acceptable for the first implementation; match-result persistence is not required.

## Acceptance Criteria

- Default match timing starts with 20 seconds of AUTO.
- Cheesy Arena always knows the current 2026 shift segment during TELEOP.
- Manual scoring tablets clearly show the current shift and whether that panel's alliance is active.
- Manual scoring remains advisory during inactive shifts; the UI communicates state but does not block input.
- The AUTO fuel winner determines Shift 1/3 vs Shift 2/4 correctly.
- Fuel entered during the 3-second pause is included in AUTO winner determination.
- Tied AUTO fuel is resolved by random alliance selection at TELEOP start.
- During the 3-second pause, displays show the shift seed as `PENDING`.
- All relevant displays show the current shift and active alliance information.
- Driver stations receive the correct single-character game-specific message for the current match.
- Tests cover timing boundaries, winner selection, DS game data, and tablet gating.
