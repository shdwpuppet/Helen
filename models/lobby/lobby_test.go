// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package lobby_test

import (
	"testing"

	db "github.com/TF2Stadium/Helen/database"
	_ "github.com/TF2Stadium/Helen/helpers"
	"github.com/TF2Stadium/Helen/internal/testhelpers"
	"github.com/TF2Stadium/Helen/models/chat"
	"github.com/TF2Stadium/Helen/models/gameserver"
	. "github.com/TF2Stadium/Helen/models/lobby"
	"github.com/TF2Stadium/Helen/models/lobby/format"
	. "github.com/TF2Stadium/Helen/models/player"
	"github.com/stretchr/testify/assert"
)

func init() {
	testhelpers.CleanupDB()
}

func TestDeleteUnusedServerRecords(t *testing.T) {
	var count int

	lobby := testhelpers.CreateLobby()
	lobby.Close(false, true)
	db.DB.Save(&gameserver.ServerRecord{})

	DeleteUnusedServers()

	err := db.DB.Model(&gameserver.ServerRecord{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Zero(t, count)
}

func TestLobbyCreation(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	lobby2, _ := GetLobbyByIDServer(lobby.ID)

	assert.Equal(t, lobby.ID, lobby2.ID)
	assert.Equal(t, lobby.ServerInfo.Host, lobby2.ServerInfo.Host)
	assert.Equal(t, lobby.ServerInfo.ID, lobby2.ServerInfo.ID)

	lobby.MapName = "cp_granary"
	lobby.Save()

	db.DB.First(lobby2)
	assert.Equal(t, "cp_granary", lobby2.MapName)
}

func TestLobbyClose(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	lobby.Save()

	req := &Requirement{
		LobbyID: lobby.ID,
	}

	var players []*Player

	for i := 0; i < 12; i++ {
		p := testhelpers.CreatePlayer()
		players = append(players, p)
		err := lobby.AddPlayer(p, i, "")
		assert.NoError(t, err)
	}

	req.Save()
	lobby.Close(false, true)
	for _, p := range players {
		db.DB.Preload("Stats").First(p, p.ID)
		assert.Equal(t, p.Stats.TotalLobbies(), 1)
	}

	var count int

	db.DB.Model(&gameserver.ServerRecord{}).Where("id = ?", lobby.ServerInfoID).Count(&count)
	assert.Zero(t, count)
	lobby, _ = GetLobbyByID(lobby.ID)
	assert.Equal(t, lobby.State, Ended)
}

func TestLobbyAdd(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	var players []*Player

	for i := 0; i < 12; i++ {
		player := testhelpers.CreatePlayer()
		players = append(players, player)
	}

	// add player
	err := lobby.AddPlayer(players[0], 0, "")
	assert.Nil(t, err)

	slot, err2 := lobby.GetPlayerSlot(players[0])
	assert.Zero(t, slot)
	assert.Nil(t, err2)

	id, err3 := lobby.GetPlayerIDBySlot(0)
	assert.Equal(t, id, players[0].ID)
	assert.Nil(t, err3)

	// try to switch slots
	err = lobby.AddPlayer(players[0], 1, "")
	assert.Nil(t, err)

	slot, err2 = lobby.GetPlayerSlot(players[0])
	assert.Equal(t, slot, 1)
	assert.Nil(t, err2)

	// this should be empty now
	id, err3 = lobby.GetPlayerIDBySlot(0)
	assert.NotNil(t, err3)

	// try to add a second player to the same slot
	err = lobby.AddPlayer(players[1], 1, "")
	assert.NotNil(t, err)

	// try to add a player to a wrong slot slot
	err = lobby.AddPlayer(players[2], 55, "")
	assert.NotNil(t, err)

	lobby2 := testhelpers.CreateLobby()
	defer lobby2.Close(false, true)
	lobby2.Save()

	// try to add a player while they're in another lobby
	//player should be substituted
	lobby.State = InProgress
	lobby.Save()
	err = lobby2.AddPlayer(players[0], 1, "")
	assert.Nil(t, err)

	var count int
	db.DB.Model(&LobbySlot{}).Where("lobby_id = ? AND needs_sub = ?", lobby.ID, true).Count(&count)
	assert.Equal(t, count, 1)
}

func TestLobbyRemove(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	player := testhelpers.CreatePlayer()

	// add player
	err := lobby.AddPlayer(player, 0, "")
	assert.Nil(t, err)

	// remove player
	err = lobby.RemovePlayer(player)
	assert.Nil(t, err)

	// this should be empty now
	_, err2 := lobby.GetPlayerIDBySlot(0)
	assert.NotNil(t, err2)

	// can add player again
	err = lobby.AddPlayer(player, 0, "")
	assert.Nil(t, err)
}

func TestLobbyBan(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	player := testhelpers.CreatePlayer()

	// add player
	err := lobby.AddPlayer(player, 0, "")
	assert.Nil(t, err)

	// ban player
	err = lobby.RemovePlayer(player)
	lobby.BanPlayer(player)
	assert.Nil(t, err)

	// should not be able to add again
	err = lobby.AddPlayer(player, 5, "")
	assert.NotNil(t, err)
}

func TestReadyPlayer(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")

	lobby.ReadyPlayer(player)
	ready, err := lobby.IsPlayerReady(player)
	assert.Equal(t, ready, true)
	assert.Nil(t, err)

	lobby.UnreadyPlayer(player)
	lobby.ReadyPlayer(player)
	ready, err = lobby.IsPlayerReady(player)
	assert.Equal(t, ready, true)
	assert.Nil(t, err)
}

func TestSetInGame(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")
	lobby.SetInGame(player)

	slot, err := lobby.GetPlayerSlotObj(player)
	assert.Nil(t, err)
	assert.True(t, slot.InGame)
}

func TestSetNotInGame(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")
	lobby.SetInGame(player)
	lobby.SetNotInGame(player)

	slot, err := lobby.GetPlayerSlotObj(player)
	assert.Nil(t, err)
	assert.False(t, slot.InGame)
}

func TestIsPlayerInGame(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")
	lobby.SetInGame(player)

	ingame := lobby.IsPlayerInGame(player)
	assert.True(t, ingame)
}

func TestIsEveryoneReady(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")
	lobby.ReadyPlayer(player)
	assert.Equal(t, lobby.IsEveryoneReady(), false)

	for i := 1; i < 12; i++ {
		player := testhelpers.CreatePlayer()
		lobby.AddPlayer(player, i, "")
		lobby.ReadyPlayer(player)
	}
	assert.Equal(t, lobby.IsEveryoneReady(), true)
}

func TestUnreadyPlayer(t *testing.T) {
	t.Parallel()
	player := testhelpers.CreatePlayer()

	player.Save()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()
	lobby.AddPlayer(player, 0, "")

	lobby.ReadyPlayer(player)
	lobby.UnreadyPlayer(player)
	ready, err := lobby.IsPlayerReady(player)
	assert.Equal(t, ready, false)
	assert.Nil(t, err)
}

func TestSpectators(t *testing.T) {
	t.Parallel()

	player := testhelpers.CreatePlayer()

	player2 := testhelpers.CreatePlayer()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	err := lobby.AddSpectator(player)
	assert.Nil(t, err)

	var specs []Player
	db.DB.Model(lobby).Association("Spectators").Find(&specs)
	assert.Equal(t, 1, len(specs))

	err = lobby.AddSpectator(player2)
	assert.Nil(t, err)

	specs = nil
	db.DB.Model(lobby).Association("Spectators").Find(&specs)
	assert.Equal(t, 2, len(specs))
	assert.Equal(t, true, specs[0].IsSpectatingID(lobby.ID))

	err = lobby.RemoveSpectator(player, false)
	assert.Nil(t, err)

	specs = nil
	db.DB.Model(lobby).Association("Spectators").Find(&specs)
	assert.Equal(t, 1, len(specs))

	// adding the same player again should not increase the count
	err = lobby.AddSpectator(player2)
	specs = nil
	db.DB.Model(lobby).Association("Spectators").Find(&specs)
	assert.Equal(t, 1, len(specs))

	// adding a player should remove them from spectators
	lobby.AddPlayer(player2, 11, "")
	specs = nil
	db.DB.Model(lobby).Association("Spectators").Find(&specs)
	assert.Zero(t, len(specs))
}

func TestUnreadyAllPlayers(t *testing.T) {
	t.Parallel()

	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	for i := 0; i < 12; i++ {
		player := testhelpers.CreatePlayer()
		lobby.AddPlayer(player, i, "")
		lobby.ReadyPlayer(player)
	}

	err := lobby.UnreadyAllPlayers()
	assert.Nil(t, err)
	ready := lobby.IsEveryoneReady()
	assert.Equal(t, ready, false)
}

func TestRemoveUnreadyPlayers(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	lobby.Save()

	var players []*Player
	for i := 0; i < 12; i++ {
		player := testhelpers.CreatePlayer()

		lobby.AddPlayer(player, i, "")
		players = append(players, player)
	}

	err := lobby.RemoveUnreadyPlayers(true)
	assert.Nil(t, err)

	for i := 0; i < 12; i++ {
		var count int
		_, err := lobby.GetPlayerIDBySlot(i)
		assert.Error(t, err)

		db.DB.Table("spectators_players_lobbies").Where("lobby_id = ? AND player_id = ?", lobby.ID, players[i].ID).Count(&count)
		assert.Equal(t, count, 1)
	}
}

func TestUpdateStats(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, false)
	var players []*Player

	for i := 0; i < 6; i++ {
		players = append(players, testhelpers.CreatePlayer())
	}
	for i, player := range players {
		err := lobby.AddPlayer(player, i, "")
		assert.NoError(t, err)
	}

	lobby.UpdateStats()
	for _, player := range players {
		db.DB.Preload("Stats").First(player, player.ID)
		assert.Equal(t, player.Stats.PlayedSixesCount, 1)
	}
}

func TestSlotRequirements(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	player := testhelpers.CreatePlayer()
	req := &Requirement{
		LobbyID: lobby.ID,
		Slot:    0,
		Hours:   1,
		Lobbies: 1,
	}
	req.Save()

	assert.True(t, lobby.HasSlotRequirement(0))
	err := lobby.AddPlayer(player, 0, "")
	assert.Equal(t, err, ErrReqHours)

	player.GameHours = 2
	player.Save()

	err = lobby.AddPlayer(player, 0, "")
	assert.Equal(t, err, ErrReqLobbies)

	player, _ = GetPlayerWithStats(player.SteamID)
	player.Stats.PlayedCountIncrease(lobby.Type)

	err = lobby.AddPlayer(player, 0, "")
	assert.NoError(t, err)

	//Adding a player to another slot shouldn't return any errors
	// req = &Requirement{
	// 	LobbyID: lobby.ID,
	// 	Slot:    -1,
	// 	Hours:   1,
	// 	Lobbies: 1,
	// }
	player2 := testhelpers.CreatePlayer()
	err = lobby.AddPlayer(player2, 1, "")
	assert.NoError(t, err)
}

func TestHasPlayer(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	player := testhelpers.CreatePlayer()

	lobby.AddPlayer(player, 1, "")
	assert.True(t, lobby.HasPlayer(player))

	player2 := testhelpers.CreatePlayer()
	assert.False(t, lobby.HasPlayer(player2))

	lobby.RemovePlayer(player)
	assert.False(t, lobby.HasPlayer(player))
}

func TestSlotNeedsSubstitute(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	player := testhelpers.CreatePlayer()

	lobby.AddPlayer(player, 1, "")
	lobby.Substitute(player)

	assert.True(t, lobby.SlotNeedsSubstitute(1))
}

func TestFillSubstitute(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)

	player := testhelpers.CreatePlayer()

	lobby.AddPlayer(player, 1, "")
	lobby.Substitute(player)

	assert.True(t, lobby.SlotNeedsSubstitute(1))
	assert.NoError(t, lobby.FillSubstitute(1))
	assert.False(t, lobby.SlotNeedsSubstitute(1))
}

func TestStart(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)

	lobby.Start()
	assert.Equal(t, lobby.CurrentState(), InProgress)
}

func TestIsSubNeeded(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)
	player := testhelpers.CreatePlayer()
	lobby.AddPlayer(player, 1, "")

	lobby.Substitute(player)
	assert.True(t, lobby.SlotNeedsSubstitute(1))

}

func TestLobbySlots(t *testing.T) {
	t.Parallel()
	lobby := testhelpers.CreateLobby()
	defer lobby.Close(false, true)

	for i := 0; i < 12; i++ {
		p := testhelpers.CreatePlayer()
		lobby.AddPlayer(p, i, "")
	}

	assert.Len(t, lobby.GetAllSlots(), 12)
}

func TestLobbyMaxSubsClose(t *testing.T) {
	t.Parallel()

	lobby := testhelpers.CreateLobby()
	lobby.Type = format.Debug
	lobby.Save()
	p1 := testhelpers.CreatePlayer()
	p2 := testhelpers.CreatePlayer()
	lobby.AddPlayer(p1, 0, "")
	lobby.AddPlayer(p2, 1, "")
	lobby.Substitute(p1)
	lobby.Substitute(p2)

	assert.Equal(t, lobby.CurrentState(), Ended, "Lobby should be closed")
	m := &chat.ChatMessage{}
	db.DB.Model(&chat.ChatMessage{}).Where("room = ?", lobby.ID).Last(m)
	assert.Equal(t, m.Message, "Lobby closed (Too many subs).")
}
