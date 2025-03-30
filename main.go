package main

import (
	"fmt"
	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"
)

var (
	ignoredPacketIds = []uint32{
		packet.IDNetworkChunkPublisherUpdate,
		packet.IDMoveActorDelta,
		packet.IDSetActorData,
		packet.IDLevelChunk,
		packet.IDMovePlayer,
		packet.IDCurrentStructureFeature,
	}
	debugIgnoredPacketIds = []uint32{
		packet.IDChunkRadiusUpdated,
		packet.IDServerToClientHandshake,
		packet.IDPlayerList,
		packet.IDBiomeDefinitionList,
		packet.IDCraftingData,
		packet.IDStartGame,
	}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <host:ip | realm_code> [--debug]\n", os.Args[0])
		return
	}
	address := os.Args[1]
	tkn, err := GetAccountToken()
	if err != nil {
		fmt.Printf("Error getting account token: %s\n", err)
		return
	}
	if len(strings.Split(os.Args[1], ":")) == 1 {
		realmCode := address
		client := realms.NewClient(*tkn)
		realm, err := client.Realm(context.Background(), realmCode)
		if err != nil {
			fmt.Printf("Error getting realm: %s\n", err)
			return
		}

		err = AcceptInvite(context.Background(), realmCode, *tkn)
		if err != nil {
			fmt.Printf("Error accepting invite: %s\n", err)
			return
		}

		address, err = realm.Address(context.Background())
		if err != nil {
			fmt.Printf("Error getting realm address: %s\n", err)
			return
		}
	}
	fmt.Printf("\033[34mConnecting to %s...\033[0m\n", address)
	startTime := time.Now()
	ctxx, canc := context.WithTimeout(context.Background(), 10*time.Second)
	defer canc()
	unconnectedPongResponse, err := raknet.PingContext(ctxx, address)
	if err != nil {
		fmt.Printf("\033[31mError pinging %s: %s\033[0m\n", address, err)
		return
	}
	pong, err := parsePong(unconnectedPongResponse)
	if err != nil {
		fmt.Printf("\033[31mError parsing pong: %s\033[0m\n", err)
		return
	}

	fmt.Printf("\033[34m%s %s\033[0m\n", pong.Edition, pong.MOTD)
	fmt.Printf("\033[34mProtocol: %d\033[0m\n", pong.ProtocolId)
	fmt.Printf("\033[34mPlayers: %d/%d\033[0m\n", pong.PlayerCount, pong.MaxPlayerCount)
	fmt.Printf("\033[34mAddress: %s\033[0m\n", address)
	fmt.Printf("\033[34mPing time: %s\033[0m\n", time.Since(startTime).String())

	dialer := minecraft.Dialer{
		TokenSource:                *tkn,
		DisconnectOnUnknownPackets: false,
		DisconnectOnInvalidPackets: false,
		ClientData: login.ClientData{
			DeviceOS:     protocol.DeviceOrbis,
			DeviceModel:  "ps_emu",
			LanguageCode: "en_us",
			GameVersion:  pong.ProtocolVersion,
		},
		Protocol: proto{
			ProtocolVersion: pong.ProtocolId,
			Version:         pong.ProtocolVersion,
		},
		FlushRate:         20,
		EnableClientCache: false,
	}

	conn, err := dialer.DialContext(context.Background(), "raknet", address)
	if err != nil {
		fmt.Printf("\033[31mError connecting to %s: %s\033[0m\n", address, err)
		return
	}
	fmt.Printf("\033[34mConnected to %s in %s\033[0m\n", address, time.Since(startTime).String())
	startTime = time.Now()
	defer func() {
		_ = conn.WritePacket(&packet.Login{})
		if err := conn.Close(); err != nil {
			fmt.Printf("\033[31mError closing connection: %s\033[0m\n", err)
			return
		}
		fmt.Printf("\033[34mDisconnected from %s after %s\033[0m\n", address, time.Since(startTime).String())
	}()
	go func() {
		fmt.Printf("\033[34mSending packets...\033[0m\n")
		for {
			_, err := conn.ReadPacket()
			if err != nil {
				fmt.Printf("\033[31mError reading packet: %s\033[0m\n", err)
				return
			}
		}
	}()

	gameData := conn.GameData()
	x, y, z := gameData.PlayerPosition.X(), gameData.PlayerPosition.Y(), gameData.PlayerPosition.Z()
	fmt.Printf("\033[34mPosition: %v, %v, %v\033[0m\n", x, y, z)
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	for {
		select {
		case <-sigint:
			return
		default:
			time.Sleep(time.Second)
		}
	}
}

func (p proto) ConvertToLatest(pk packet.Packet, _ *minecraft.Conn) []packet.Packet {
	if slices.Contains(ignoredPacketIds, pk.ID()) {
		return nil
	}
	if slices.Contains(os.Args, "--debug") && !slices.Contains(debugIgnoredPacketIds, pk.ID()) {
		fmt.Printf("\033[36mIncoming packet %T %+v\033[0m\n", pk, pk)
	}

	switch packetV := pk.(type) {
	case *packet.ResourcePacksInfo:
		return []packet.Packet{&packet.ResourcePacksInfo{
			TexturePackRequired: false,
			HasAddons:           false,
			HasScripts:          false,
			TexturePacks:        nil,
		}}
	case *packet.ResourcePackStack:
		fmt.Printf("\033[34mResourcePackStack: %v %v %v\033[0m\n", packetV.BehaviourPacks, packetV.TexturePacks, packetV.Experiments)
	case *packet.LevelEventGeneric:
		fmt.Printf("\033[34mLevelEventGeneric: %v %v\033[0m\n", packetV.EventID, string(packetV.SerialisedEventData))
	}
	return []packet.Packet{pk}
}
func (p proto) ConvertFromLatest(pk packet.Packet, _ *minecraft.Conn) []packet.Packet {
	if slices.Contains(ignoredPacketIds, pk.ID()) {
		return nil
	}
	if slices.Contains(os.Args, "--debug") && !slices.Contains(debugIgnoredPacketIds, pk.ID()) {
		fmt.Printf("\033[36mOutgoing packet %T\033[0m\n", pk)
	}

	switch packetV := pk.(type) {
	case *packet.ResourcePacksInfo:
		return []packet.Packet{&packet.ResourcePacksInfo{
			TexturePackRequired: false,
			HasAddons:           false,
			HasScripts:          false,
			TexturePacks:        nil,
		}}
	case *packet.Login:
		_, clientData, _, err := login.Parse(packetV.ConnectionRequest)
		if err != nil {
			fmt.Printf("\033[31mError parsing login packet: %s\033[0m\n", err)
			break
		}
		fmt.Printf("\033[34mThirdPartyName: %v\033[0m\n", clientData.ThirdPartyNameOnly)
		break
	}

	return []packet.Packet{pk}
}

func AcceptInvite(ctx context.Context, code string, tkn oauth2.TokenSource) error {
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://pocket.realms.minecraft.net/invites/v1/link/accept/%s", code), nil)
	for k, v := range map[string]string{
		"Accept":                   "*/*",
		"charset":                  "utf-8",
		"client-ref":               "1dbf893ab5ebfb96af356e196cf516e0e4596fb0",
		"client-version":           "1.21.50",
		"user-agent":               "MCPE/UWP",
		"x-clientplatform":         "Windows",
		"x-networkprotocolversion": "766",
		"Accept-Language":          "en-CA",
		"Accept-Encoding":          "gzip, deflate, br",
		"Host":                     "pocket.realms.minecraft.net",
		"Content-Length":           "0",
		"Connection":               "Keep-Alive",
		"Cache-Control":            "no-cache",
	} {
		req.Header.Set(k, v)
	}
	tknV, err := tkn.Token()
	if err != nil {
		return err
	}
	tknV.SetAuthHeader(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP Error: %d", resp.StatusCode)
	}

	return nil
}
