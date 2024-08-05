package msrv

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/go-tpm/legacy/tpm2"
	"github.com/google/go-tpm/tpmutil"
	etpm "github.com/lf-edge/eve/pkg/pillar/evetpm"
)

type ActivateCredTpmParam struct {
	Ek      string `json:"ek"`
	AikPub  string `json:"aikpub"`
	AikName string `json:"aikname"`
}

type ActivateCredGenerated struct {
	Cred   string `json:"cred"`
	Secret string `json:"secret"`
}

type ActivateCredActivated struct {
	Secret string `json:"secret"`
	Digest string `json:"digest"`
	Sig    string `json:"sig"`
}

// handles the GET request /tmp/activate-credential/, this is used to get the
// HTPM EK public key, HWTPM AIK public key, HWTPM AIK name and SWTPM EK signature.
func getActivateCredntialParams(uuid string) ([]byte, []byte, []byte, error) {
	rw, err := tpm2.OpenTPM(etpm.TpmDevicePath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rw.Close()

	ekPub, _, _, err := tpm2.ReadPublic(rw, etpm.TpmEKHdl)
	if err != nil {
		return nil, nil, nil, err
	}

	ekPubByte, err := ekPub.Encode()
	if err != nil {
		return nil, nil, nil, err
	}

	var aikName tpmutil.U16Bytes
	aikPub, aikName, _, err := tpm2.ReadPublic(rw, etpm.TpmAIKHdl)
	if err != nil {
		return nil, nil, nil, err
	}

	aikPubByte, err := aikPub.Encode()
	if err != nil {
		return nil, nil, nil, err
	}

	aikNameMarshaled := &bytes.Buffer{}
	if err := aikName.TPMMarshal(aikNameMarshaled); err != nil {
		return nil, nil, nil, err
	}

	return ekPubByte, aikPubByte, aikNameMarshaled.Bytes(), nil
}

// handles the POST request /tmp/activate-credential/, this is used to activate
// the credential (decrypt the secret) using HWTPM EK and HWTPM AIK.
func activateCredntial(uuid string, jsonData []byte) ([]byte, []byte, []byte, error) {
	rw, err := tpm2.OpenTPM(etpm.TpmDevicePath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rw.Close()

	var credPayload ActivateCredGenerated
	if err := json.Unmarshal(jsonData, &credPayload); err != nil {
		return nil, nil, nil, err
	}

	credBlob, err := base64.StdEncoding.DecodeString(credPayload.Cred)
	if err != nil {
		return nil, nil, nil, err
	}

	encryptedSecret, err := base64.StdEncoding.DecodeString(credPayload.Secret)
	if err != nil {
		return nil, nil, nil, err
	}

	// we need to skip the first 2 bytes of the credBlob and encryptedSecret
	// as it contains the type. so make sure the length is greater than 2.
	if len(credBlob) < 2 || len(encryptedSecret) < 2 {
		return nil, nil, nil, fmt.Errorf("malformed parameters")
	}
	credBlob = credBlob[2:]
	encryptedSecret = encryptedSecret[2:]

	// activate the credential
	recoveredCred, err := tpm2.ActivateCredential(rw,
		etpm.TpmAIKHdl,
		etpm.TpmEKHdl,
		etpm.EmptyPassword,
		etpm.EmptyPassword,
		credBlob,
		encryptedSecret)
	if err != nil {
		return nil, nil, nil, err
	}

	// FIX-ME : this should be SWTP EK based on the uuid
	data := []byte("some data to sign")
	digest, validation, err := tpm2.Hash(rw, tpm2.AlgSHA256, data, tpm2.HandleOwner)
	if err != nil {
		return nil, nil, nil, err
	}

	sig, err := tpm2.Sign(rw, etpm.TpmAIKHdl, etpm.EmptyPassword, digest, validation, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	return recoveredCred, digest, sig.RSA.Signature, nil
}
