// Copyright (c) 2018-2020 The asimov developers
// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package protos

import (
	"fmt"
	"github.com/AsimovNetwork/asimov/common"
	"io"

	"github.com/AsimovNetwork/asimov/common/serialization"
)

// MaxBlockLocatorsPerMsg is the maximum number of block locator hashes allowed
// per message.
const MaxBlockLocatorsPerMsg = 500

// MsgGetBlocks implements the Message interface and represents a bitcoin
// getblocks message.  It is used to request a list of blocks starting after the
// last known hash in the slice of block locator hashes.  The list is returned
// via an inv message (MsgInv) and is limited by a specific hash to stop at or
// the maximum number of blocks per message, which is currently 500.
//
// Set the HashStop field to the hash at which to stop and use
// AddBlockLocatorHash to build up the list of block locator hashes.
//
// The algorithm for building the block locator hashes should be to add the
// hashes in reverse order until you reach the genesis block.  In order to keep
// the list of locator hashes to a reasonable number of entries, first add the
// most recent 10 block hashes, then double the step each loop iteration to
// exponentially decrease the number of hashes the further away from head and
// closer to the genesis block you get.
type MsgGetBlocks struct {
	ProtocolVersion    uint32
	BlockLocatorHashes []*common.Hash
	HashStop           common.Hash
}

// AddBlockLocatorHash adds a new block locator hash to the message.
func (msg *MsgGetBlocks) AddBlockLocatorHash(hash *common.Hash) error {
	if len(msg.BlockLocatorHashes)+1 > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message [max %v]",
			MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.AddBlockLocatorHash", str)
	}

	msg.BlockLocatorHashes = append(msg.BlockLocatorHashes, hash)
	return nil
}

// VVSDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetBlocks) VVSDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := serialization.ReadUint32(r, &msg.ProtocolVersion)
	if err != nil {
		return err
	}

	// Read num block locator hashes and limit to max.
	count, err := serialization.ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.VVSDecode", str)
	}

	// Create a contiguous slice of hashes to deserialize into in order to
	// reduce the number of allocations.
	locatorHashes := make([]common.Hash, count)
	msg.BlockLocatorHashes = make([]*common.Hash, 0, count)
	for i := uint64(0); i < count; i++ {
		hash := &locatorHashes[i]
		err := serialization.ReadNBytes(r, hash[:], common.HashLength)
		if err != nil {
			return err
		}
		err = msg.AddBlockLocatorHash(hash)
		if err != nil {
			return err
		}
	}

	return serialization.ReadNBytes(r, msg.HashStop[:], common.HashLength)
}

// VVSEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetBlocks) VVSEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	count := len(msg.BlockLocatorHashes)
	if count > MaxBlockLocatorsPerMsg {
		str := fmt.Sprintf("too many block locator hashes for message "+
			"[count %v, max %v]", count, MaxBlockLocatorsPerMsg)
		return messageError("MsgGetBlocks.VVSEncode", str)
	}

	err := serialization.WriteUint32(w, msg.ProtocolVersion)
	if err != nil {
		return err
	}

	err = serialization.WriteVarInt(w, pver, uint64(count))
	if err != nil {
		return err
	}

	for _, hash := range msg.BlockLocatorHashes {
		err = serialization.WriteNBytes(w, hash[:])
		if err != nil {
			return err
		}
	}

	return serialization.WriteNBytes(w, msg.HashStop[:])
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetBlocks) Command() string {
	return CmdGetBlocks
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetBlocks) MaxPayloadLength(pver uint32) uint32 {
	// Protocol version 4 bytes + num hashes (varInt) + max block locator
	// hashes + hash stop.
	return 4 + serialization.MaxVarIntPayload + (MaxBlockLocatorsPerMsg * common.HashLength) + common.HashLength
}

// NewMsgGetBlocks returns a new bitcoin getblocks message that conforms to the
// Message interface using the passed parameters and defaults for the remaining
// fields.
func NewMsgGetBlocks(hashStop *common.Hash) *MsgGetBlocks {
	return &MsgGetBlocks{
		ProtocolVersion:    common.ProtocolVersion,
		BlockLocatorHashes: make([]*common.Hash, 0, MaxBlockLocatorsPerMsg),
		HashStop:           *hashStop,
	}
}
