// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package handler

import (
	"errors"

	"github.com/TF2Stadium/Helen/models/lobby"
	"github.com/TF2Stadium/wsevent"
	"github.com/bitly/go-simplejson"
)

type Global struct{}

func (Global) Name(s string) string {
	return string((s[0])+32) + s[1:]
}

func (Global) GetConstant(so *wsevent.Client, args struct {
	Constant string `json:"constant"`
}) interface{} {

	output := simplejson.New()
	switch args.Constant {
	case "lobbySettingsList":
		output = lobby.LobbySettingsToJSON()
	default:
		return errors.New("Unknown constant.")
	}

	return newResponse(output)
}

// func (Global) GetSocketInfo(so *wsevent.Client, data []byte) interface{} {
// 	socketinfo := struct {
// 		ID    string   `json:"id"`
// 		Rooms []string `json:"rooms"`
// 	}{so.Id(), server.RoomsJoined(so.Id())}

// 	return chelpers.NewResponse(socketinfo)
// }
