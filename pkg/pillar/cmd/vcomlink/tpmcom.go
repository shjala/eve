// Copyright (c) 2018-2024 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package vcomlink

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/go-tpm/legacy/tpm2"
	etpm "github.com/lf-edge/eve/pkg/pillar/evetpm"
	"github.com/lf-edge/eve/pkg/pillar/vcom"
)

func handleTPMCom(data []byte) ([]byte, error) {
	var tpmPacket vcom.TpmRequest
	if err := json.Unmarshal(data, &tpmPacket); err != nil {
		return nil, fmt.Errorf("unable to unmarshal data: %w", err)
	}

	switch tpmPacket.Request {
	case vcom.RequestTpmGetEk:
		ek, err := getEkPub()
		if err != nil {
			return nil, fmt.Errorf("unable to get EK public key: %w", err)
		}
		return ek, nil
	default:
		return nil, fmt.Errorf("unknown request: %d", tpmPacket.Request)
	}
}

func getEkPub() ([]byte, error) {
	rw, err := tpm2.OpenTPM(etpm.TpmDevicePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open TPM: %w", err)
	}
	defer rw.Close()

	ek, _, _, err := tpm2.ReadPublic(rw, etpm.TpmEKHdl)
	if err != nil {
		return nil, fmt.Errorf("unable to read EK public: %w", err)
	}
	ekBytes, err := ek.Encode()
	if err != nil {
		return nil, fmt.Errorf("unable to encode EK public: %w", err)
	}
	ekJSON, err := json.Marshal(vcom.TpmResponseEk{
		Ek: base64.StdEncoding.EncodeToString(ekBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to marshal EK public: %w", err)
	}

	return ekJSON, nil
}
