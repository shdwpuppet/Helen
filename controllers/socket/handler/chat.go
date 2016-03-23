// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	chelpers "github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	db "github.com/TF2Stadium/Helen/database"
	"github.com/TF2Stadium/Helen/helpers"
	"github.com/TF2Stadium/Helen/models"
	"github.com/TF2Stadium/wsevent"
)

type Chat struct{}

func (Chat) Name(s string) string {
	return string((s[0])+32) + s[1:]
}

func (Chat) ChatSend(so *wsevent.Client, args struct {
	Message *string `json:"message"`
	Room    *int    `json:"room"`
}) interface{} {
	player := chelpers.GetPlayer(so.Token)
	if banned, until := player.IsBannedWithTime(models.PlayerBanChat); banned {
		ban, _ := player.GetActiveBan(models.PlayerBanChat)
		return fmt.Errorf("You've been banned from creating lobbies till %s (%s)", until.Format(time.RFC822), ban.Reason)
	}

	if *args.Room > 0 {
		var count int
		spec := player.IsSpectatingID(uint(*args.Room))
		//Check if player has either joined, or is spectating lobby
		db.DB.Table("lobby_slots").Where("lobby_id = ? AND player_id = ?", *args.Room, player.ID).Count(&count)

		if !spec && count == 0 {
			return errors.New("Player is not in the lobby.")
		}
	} else {
		// else room is the lobby list room
		*args.Room = 0
	}
	switch {
	case len(*args.Message) == 0:
		return errors.New("Cannot send an empty message")

	case (*args.Message)[0] == '\n':
		return errors.New("Cannot send messages prefixed with newline")

	case len(*args.Message) > 150:
		return errors.New("Message too long")
	}

	message := models.NewChatMessage(*args.Message, *args.Room, player)

	if strings.HasPrefix(*args.Message, "!admin") {
		chelpers.SendToSlack(*args.Message, player.Name, player.SteamID)
		return emptySuccess
	}

	message.Save()
	message.Send()

	return emptySuccess
}

func (Chat) ChatDelete(so *wsevent.Client, args struct {
	ID   *int  `json:"id"`
	Room *uint `json:"room"`
}) interface{} {

	if err := chelpers.CheckPrivilege(so, helpers.ActionDeleteChat); err != nil {
		return err
	}

	message := &models.ChatMessage{}
	err := db.DB.First(message, *args.ID).Error
	if message.Bot {
		return errors.New("Cannot delete notification messages")
	}
	if err != nil {
		return errors.New("Can't find message")
	}

	message.Deleted = true
	message.Save()
	message.Send()

	return emptySuccess
}
