// Copyright (c) 2018-2024 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package vcom

// BasePacket is the base packet for all other packets
// it should be embedded in other packets.
type BasePacket struct {
	Channel uint `json:"channel"`
}

// TpmRequest is the request packet for TPM related requests
type TpmRequest struct {
	BasePacket
	Request uint `json:"request"`
}

// ErrorResponse is the response packet for errors
type ErrorResponse struct {
	Error string `json:"error"`
}

// TpmResponseEk is the response packet for TPM Endorsement Key
type TpmResponseEk struct {
	Ek string `json:"ek"`
}

const (
	baseChannelID = 1
	// HostVPort is the port on which the host listens
	HostVPort = 2000
	// List of available channel, channel name should be in form of
	// Channel<Name> = baseChannelID + X
	//
	//ChannelTpm is the channel for TPM related requests
	ChannelTpm = baseChannelID + 1
	// List of available requests, request name should be in form of
	// Request<ChannelName><Request> = X
	//
	//RequestTpmGetEk is the request to get the TPM Endorsement Key
	RequestTpmGetEk = 1
)
