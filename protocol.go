package main

import (
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type proto struct {
	ProtocolVersion int32
	Version         string
	shieldID        int32
}

func (p proto) ID() int32   { return p.ProtocolVersion }
func (p proto) Ver() string { return p.Version }
func (p proto) Packets(listener bool) packet.Pool {
	if listener {
		return packet.NewClientPool()
	}
	return packet.NewServerPool()
}
func (p proto) NewReader(r minecraft.ByteReader, shieldID int32, enableLimits bool) protocol.IO {
	p.shieldID = shieldID
	return protocol.NewReader(r, shieldID, enableLimits)
}
func (p proto) NewWriter(w minecraft.ByteWriter, shieldID int32) protocol.IO {
	p.shieldID = shieldID
	return protocol.NewWriter(w, shieldID)
}
