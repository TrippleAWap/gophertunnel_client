package main

import (
	"fmt"
	"strconv"
)

type unconnectedPong struct {
	Edition         string
	MOTD            string
	ProtocolId      int32
	ProtocolVersion string
	PlayerCount     int32
	MaxPlayerCount  int32
	ServerUUID      string
	GameMode        string
	GameModeId      int32
	IPv4Port        int32
	IPv6Port        int32
}

func splitPong(s string) []string {
	var runes []rune
	var tokens []string
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\\':
			inEscape = true
		case r == ';':
			tokens = append(tokens, string(runes))
			runes = runes[:0]
		case inEscape:
			inEscape = false
			fallthrough
		default:
			runes = append(runes, r)
		}
	}
	return append(tokens, string(runes))
}

func parsePong(pong []byte) (*unconnectedPong, error) {
	data := splitPong(string(pong))
	if len(data) < 11 {
		data = append(data, "19132", "19132")
	}
	protocolId, err := strconv.Atoi(data[2])
	if err != nil {
		return nil, fmt.Errorf("invalid protocol id: %s", data[2])
	}
	playerCount, err := strconv.Atoi(data[4])
	if err != nil {
		return nil, fmt.Errorf("invalid player count: %s", data[4])
	}
	maxPlayerCount, err := strconv.Atoi(data[5])
	if err != nil {
		return nil, fmt.Errorf("invalid max player count: %s", data[5])
	}
	gameModeId, err := strconv.Atoi(data[9])
	if err != nil {
		gameModeId = 0
	}
	ipv4Port, err := strconv.Atoi(data[10])
	if err != nil {
		return nil, fmt.Errorf("invalid ipv4 port: %s", data[10])
	}
	ipv6Port, err := strconv.Atoi(data[11])
	if err != nil {
		return nil, fmt.Errorf("invalid ipv6 port: %s", data[11])
	}
	return &unconnectedPong{
		Edition:         data[0],
		MOTD:            data[1],
		ProtocolId:      int32(protocolId),
		ProtocolVersion: data[3],
		PlayerCount:     int32(playerCount),
		MaxPlayerCount:  int32(maxPlayerCount),
		ServerUUID:      data[6],
		GameMode:        data[8],
		GameModeId:      int32(gameModeId),
		IPv4Port:        int32(ipv4Port),
		IPv6Port:        int32(ipv6Port),
	}, nil
}
