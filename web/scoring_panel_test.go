// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package web

import (
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func setTestScoringPanelConfig(t *testing.T) {
	previousConfig := game.ActiveGameConfig
	game.ActiveGameConfig = &game.GameConfigDefinition{
		Scoring: []game.ScoringElement{
			{Id: "fuel", Label: "Fuel", PointValue: 1, CountsForAutoFuel: true},
		},
		Panels: []game.PanelConfig{
			{
				Id:    "red_near",
				Title: "Red Near",
				Widgets: []game.WidgetConfig{
					{Id: "red_counter", Type: "counter", Label: "Fuel", ScoringId: "fuel"},
				},
			},
			{
				Id:    "blue_near",
				Title: "Blue Near",
				Widgets: []game.WidgetConfig{
					{Id: "blue_toggle", Type: "toggle", Label: "Fuel", ScoringId: "fuel"},
				},
			},
			{Id: "red_far", Title: "Red Far"},
			{Id: "blue_far", Title: "Blue Far"},
		},
	}
	t.Cleanup(func() {
		game.ActiveGameConfig = previousConfig
	})
}

func TestScoringPanel(t *testing.T) {
	web := setupTestWeb(t)
	setTestScoringPanelConfig(t)

	recorder := web.getHttpResponse("/panels/scoring/invalidposition")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Scoring Panel - Untitled Event - Cheesy Arena")
	recorder = web.getHttpResponse("/panels/scoring/red_near")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/red_far")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/blue_near")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/blue_far")
	assert.Equal(t, 200, recorder.Code)
}

func TestScoringPanelWebsocket(t *testing.T) {
	web := setupTestWeb(t)
	setTestScoringPanelConfig(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()

	_, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blorpy/websocket", nil)
	assert.NotNil(t, err)

	redConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/red_near/websocket", nil)
	assert.Nil(t, err)
	defer redConn.Close()
	redWs := websocket.NewTestWebsocket(redConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red_near"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumPanels("blue_near"))

	blueConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blue_near/websocket", nil)
	assert.Nil(t, err)
	defer blueConn.Close()
	blueWs := websocket.NewTestWebsocket(blueConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red_near"))
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("blue_near"))

	// Should get a few status updates right after connection.
	readWebsocketType(t, redWs, "resetLocalState")
	readWebsocketType(t, redWs, "matchLoad")
	readWebsocketType(t, redWs, "matchTime")
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "resetLocalState")
	readWebsocketType(t, blueWs, "matchLoad")
	readWebsocketType(t, blueWs, "matchTime")
	readWebsocketType(t, blueWs, "realtimeScore")

	redWs.Write("widget", map[string]any{"widgetId": "red_counter", "delta": 1})
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "realtimeScore")
	assert.Equal(t, 1, web.arena.RedRealtimeScore.CurrentScore.GenericCounters["red_counter"])

	blueWs.Write("widget", map[string]any{"widgetId": "blue_toggle"})
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "realtimeScore")
	assert.True(t, web.arena.BlueRealtimeScore.CurrentScore.GenericToggles["blue_toggle"])

	redWs.Write("addFoul", map[string]any{"alliance": "blue", "isMajor": true})
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "realtimeScore")
	if assert.Equal(t, 1, len(web.arena.BlueRealtimeScore.CurrentScore.Fouls)) {
		assert.True(t, web.arena.BlueRealtimeScore.CurrentScore.Fouls[0].IsMajor)
	}

	arenaMatchStateBeforeCommit := web.arena.MatchState
	web.arena.MatchState = field.AutoPeriod
	redWs.Write("commitMatch", nil)
	readWebsocketType(t, redWs, "error")
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red_near"))

	web.arena.MatchState = field.PostMatch
	redWs.Write("commitMatch", nil)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red_near"))
	web.arena.MatchState = arenaMatchStateBeforeCommit
}
