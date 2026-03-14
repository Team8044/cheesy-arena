// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Shared client-side logic for interpreting match state and timing notifications.

// MatchType enum values.
const matchTypeTest = 0;
const matchTypePractice = 1;
const matchTypeQualification = 2;
const matchTypePlayoff = 3;

const matchStates = {
  0: "PRE_MATCH",
  1: "START_MATCH",
  2: "WARMUP_PERIOD",
  3: "AUTO_PERIOD",
  4: "PAUSE_PERIOD",
  5: "TELEOP_PERIOD",
  6: "POST_MATCH",
  7: "TIMEOUT_ACTIVE",
  8: "POST_TIMEOUT"
};
let matchTiming;

// Handles a websocket message containing the length of each period in the match.
const handleMatchTiming = function (data) {
  matchTiming = data;
};

// Converts the raw match state and time into a human-readable state and per-period time. Calls the provided
// callback with the result.
const translateMatchTime = function (data, callback) {
  var matchStateText;
  switch (matchStates[data.MatchState]) {
    case "PRE_MATCH":
      matchStateText = "PRE-MATCH";
      break;
    case "START_MATCH":
    case "WARMUP_PERIOD":
      matchStateText = "WARMUP";
      break;
    case "AUTO_PERIOD":
      matchStateText = "AUTONOMOUS";
      break;
    case "PAUSE_PERIOD":
      matchStateText = "PAUSE";
      break;
    case "TELEOP_PERIOD":
      matchStateText = "TELEOPERATED";
      break;
    case "POST_MATCH":
      matchStateText = "POST-MATCH";
      break;
    case "TIMEOUT_ACTIVE":
    case "POST_TIMEOUT":
      matchStateText = "TIMEOUT";
      break;
  }
  callback(
    matchStates[data.MatchState],
    matchStateText,
    getCountdown(data.MatchState, data.MatchTimeSec),
    getShiftStatusText(data),
  );
};

// Returns the per-period countdown for the given match state and overall time into the match.
const getCountdown = function (matchState, matchTimeSec) {
  switch (matchStates[matchState]) {
    case "PRE_MATCH":
    case "START_MATCH":
    case "WARMUP_PERIOD":
      return matchTiming.AutoDurationSec;
    case "AUTO_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec - matchTimeSec;
    case "TELEOP_PERIOD":
      return matchTiming.WarmupDurationSec + matchTiming.AutoDurationSec + matchTiming.TeleopDurationSec +
        matchTiming.PauseDurationSec - matchTimeSec;
    case "TIMEOUT_ACTIVE":
      return matchTiming.TimeoutDurationSec - matchTimeSec;
    default:
      return 0;
  }
};

// Converts the given countdown in seconds to a string with a colon separator and leading zero padding.
const getCountdownString = function (countdownSec) {
  let countdownString = String(countdownSec % 60);
  if (countdownString.length === 1) {
    countdownString = "0" + countdownString;
  }
  return Math.floor(countdownSec / 60) + ":" + countdownString;
};

const getActiveAllianceText = function (data) {
  switch (data.ActiveAlliance) {
    case "RED":
      return "RED ACTIVE";
    case "BLUE":
      return "BLUE ACTIVE";
    case "BOTH":
      return "BOTH ACTIVE";
    case "PENDING":
      return "WINNER PENDING";
    default:
      return "";
  }
};

const getShiftStatusText = function (data) {
  if (!data || !data.MatchSegmentLabel) {
    return "";
  }
  const activeText = getActiveAllianceText(data);
  if (!activeText) {
    return data.MatchSegmentLabel;
  }
  return `${data.MatchSegmentLabel} - ${activeText}`;
};

const getAllianceShiftActivityText = function (data, alliance) {
  if (!data) {
    return "";
  }
  if (data.ActiveAlliance === "PENDING") {
    return "PENDING";
  }
  if (data.ActiveAlliance === "BOTH") {
    return "BOTH ACTIVE";
  }
  if (data.ActiveAlliance === "RED") {
    return alliance === "R" ? "ACTIVE" : "INACTIVE";
  }
  if (data.ActiveAlliance === "BLUE") {
    return alliance === "B" ? "ACTIVE" : "INACTIVE";
  }
  return "";
};

const getPhaseFromMatchTime = function (data) {
  const state = matchStates[data.MatchState];
  if (state === "AUTO_PERIOD" || state === "PAUSE_PERIOD") {
    return "auto";
  }
  if (state === "TELEOP_PERIOD") {
    if (data.MatchSegment === "transition") {
      return "transition";
    }
    if (data.MatchSegment === "shift1") {
      return "shift1";
    }
    if (data.MatchSegment === "shift2") {
      return "shift2";
    }
    if (data.MatchSegment === "shift3") {
      return "shift3";
    }
    if (data.MatchSegment === "shift4") {
      return "shift4";
    }
    if (data.MatchSegment === "endgame") {
      return "endgame";
    }
    return "teleop";
  }
  if (state === "POST_MATCH") {
    return "post";
  }
  return "pregame";
};
