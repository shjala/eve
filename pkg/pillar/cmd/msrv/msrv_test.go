// Copyright (c) 2024 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package msrv_test

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/go-tpm/legacy/tpm2"
	cred "github.com/google/go-tpm/legacy/tpm2/credactivation"
	"github.com/lf-edge/eve/pkg/pillar/base"
	"github.com/lf-edge/eve/pkg/pillar/cmd/msrv"
	"github.com/lf-edge/eve/pkg/pillar/pubsub"
	"github.com/lf-edge/eve/pkg/pillar/types"
	"github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

func TestPostKubeconfig(t *testing.T) {
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	logger := logrus.StandardLogger()

	log := base.NewSourceLogObject(logger, "pubsub", 1234)
	ps := pubsub.New(pubsub.NewMemoryDriver(), logger, log)

	appNetworkStatus, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "zedrouter",
		TopicType:  types.AppNetworkStatus{},
		Persistent: true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	u, err := uuid.FromString("6ba7b810-9dad-11d1-80b4-000000000000")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	err = appNetworkStatus.Publish("6ba7b810-9dad-11d1-80b4-000000000001", types.AppNetworkStatus{
		UUIDandVersion: types.UUIDandVersion{
			UUID:    u,
			Version: "1.0",
		},
		AppNetAdapterList: []types.AppNetAdapterStatus{
			{
				AllocatedIPv4Addr: net.ParseIP("192.168.1.1"),
			},
		},
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	devNetStatusPub, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "nim",
		TopicType:  types.DeviceNetworkStatus{},
		Persistent: true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	devNetStat := types.DeviceNetworkStatus{
		Ports: []types.NetworkPortStatus{
			{
				IfName: "eth0",
				AddrInfoList: []types.AddrInfo{
					{
						Addr: net.ParseIP("192.168.1.1"),
					},
				},
			},
		},
	}
	err = devNetStatusPub.Publish("6ba7b810-9dad-11d1-80b4-000000000002", devNetStat)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	netInstance, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "zedrouter",
		TopicType:  types.NetworkInstanceStatus{},
		Persistent: true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	niStatus := types.NetworkInstanceStatus{
		NetworkInstanceInfo: types.NetworkInstanceInfo{
			IPAssignments: map[string]types.AssignedAddrs{"k": {
				IPv4Addr: net.ParseIP("192.168.1.1"),
			}},
		},
		SelectedUplinkIntfName: "eth0",
	}
	err = netInstance.Publish("6ba7b810-9dad-11d1-80b4-000000000003", niStatus)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	srv := &msrv.Msrv{
		Log:    log,
		PubSub: ps,
		Logger: logger,
	}

	dir, err := ioutil.TempDir("/tmp", "msrv_test")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer os.RemoveAll(dir)

	err = srv.Init(dir, true)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	err = srv.Activate()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	handler := srv.MakeMetadataHandler()

	var jsonStr = []byte(`{"hello":"world"}`)
	descReq := httptest.NewRequest(http.MethodPost, "/eve/v1/kubeconfig", bytes.NewBuffer(jsonStr))
	descReq.Header.Set("Content-Type", "application/json")
	descReq.RemoteAddr = "192.168.1.1:0"
	descResp := httptest.NewRecorder()

	handler.ServeHTTP(descResp, descReq)
	g.Expect(descResp.Code).To(gomega.Equal(http.StatusOK))

	subPatchUsage, err := ps.NewSubscription(pubsub.SubscriptionOptions{
		AgentName:   "msrv",
		MyAgentName: "test",
		TopicImpl:   types.AppInstMetaData{},
		Activate:    true,
		Persistent:  true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	items := subPatchUsage.GetAll()
	expected := types.AppInstMetaData{
		AppInstUUID: u,
		Data:        jsonStr,
		Type:        types.AppInstMetaDataTypeKubeConfig,
	}

	g.Expect(items[expected.Key()]).To(gomega.BeEquivalentTo(expected))
}

func TestRequestPatchEnvelopes(t *testing.T) {
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	logger := logrus.StandardLogger()

	log := base.NewSourceLogObject(logger, "pubsub", 1234)
	ps := pubsub.New(pubsub.NewMemoryDriver(), logger, log)

	appNetworkStatus, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "zedrouter",
		TopicType:  types.AppNetworkStatus{},
		Persistent: true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	u, err := uuid.FromString("6ba7b810-9dad-11d1-80b4-000000000000")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	err = appNetworkStatus.Publish("6ba7b810-9dad-11d1-80b4-000000000001", types.AppNetworkStatus{
		UUIDandVersion: types.UUIDandVersion{
			UUID:    u,
			Version: "1.0",
		},
		AppNetAdapterList: []types.AppNetAdapterStatus{
			{
				AllocatedIPv4Addr: net.ParseIP("192.168.1.1"),
			},
		},
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	peInfo, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "zedagent",
		TopicType:  types.PatchEnvelopeInfoList{},
		Persistent: true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	err = peInfo.Publish("global", types.PatchEnvelopeInfoList{
		Envelopes: []types.PatchEnvelopeInfo{
			{
				Name:        "asdf",
				Version:     "asdf",
				AllowedApps: []string{"6ba7b810-9dad-11d1-80b4-000000000000"},
				PatchID:     "6ba7b810-9dad-11d1-80b4-111111111111",
				State:       types.PatchEnvelopeStateActive,
				BinaryBlobs: []types.BinaryBlobCompleted{
					{
						FileName: "abcd",
						FileSha:  "abcd",
						URL:      "a.txt",
					},
				},
			},
		},
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	srv := &msrv.Msrv{
		Log:    log,
		PubSub: ps,
		Logger: logger,
	}

	dir, err := ioutil.TempDir("/tmp", "msrv_test")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer os.RemoveAll(dir)

	err = srv.Init(dir, true)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	err = srv.Activate()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Eventually(func() []types.PatchEnvelopeInfo {
		return srv.PatchEnvelopes.Get("6ba7b810-9dad-11d1-80b4-000000000000").Envelopes
	}, 30*time.Second, 10*time.Second).Should(gomega.HaveLen(1))

	handler := srv.MakeMetadataHandler()

	descReq := httptest.NewRequest(http.MethodGet, "/eve/v1/patch/description.json", nil)
	descReq.RemoteAddr = "192.168.1.1:0"
	descResp := httptest.NewRecorder()
	descReqTimes := 42
	for i := 0; i < descReqTimes; i++ {
		handler.ServeHTTP(descResp, descReq)
		g.Expect(descResp.Code).To(gomega.Equal(http.StatusOK))

		defer descResp.Body.Reset()
		var got []msrv.PeInfoToDisplay

		err = json.NewDecoder(descResp.Body).Decode(&got)
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(got).To(gomega.BeEquivalentTo(
			[]msrv.PeInfoToDisplay{
				{
					PatchID: "6ba7b810-9dad-11d1-80b4-111111111111",
					Version: "asdf",
					BinaryBlobs: []types.BinaryBlobCompleted{
						{
							FileName:         "abcd",
							FileSha:          "abcd",
							FileMetadata:     "",
							ArtifactMetadata: "",
							URL:              "http://169.254.169.254/eve/v1/patch/download/6ba7b810-9dad-11d1-80b4-111111111111/abcd",
							Size:             0,
						},
					},
				},
			},
		))
	}

	downReq := httptest.NewRequest(http.MethodGet, "/eve/v1/patch/download/6ba7b810-9dad-11d1-80b4-111111111111/abcd", nil)
	downReq.RemoteAddr = "192.168.1.1:0"
	downResp := httptest.NewRecorder()
	downReqTimes := 24
	for i := 0; i < downReqTimes; i++ {
		handler.ServeHTTP(downResp, downReq)
		g.Expect(descResp.Code).To(gomega.Equal(http.StatusOK))
	}

	expected := types.PatchEnvelopeUsage{
		AppUUID:           "6ba7b810-9dad-11d1-80b4-000000000000",
		PatchID:           "6ba7b810-9dad-11d1-80b4-111111111111",
		Version:           "asdf",
		PatchAPICallCount: uint64(descReqTimes),
		DownloadCount:     uint64(downReqTimes),
	}

	srv.PublishPatchEnvelopesUsage()
	subPatchUsage, err := ps.NewSubscription(pubsub.SubscriptionOptions{
		AgentName:   "msrv",
		MyAgentName: "test",
		TopicImpl:   types.PatchEnvelopeUsage{},
		Activate:    true,
		Persistent:  true,
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())
	items := subPatchUsage.GetAll()
	item, ok := items["patchEnvelopeUsage:6ba7b810-9dad-11d1-80b4-111111111111-v-asdf-app-6ba7b810-9dad-11d1-80b4-000000000000"]
	g.Expect(ok).To(gomega.BeTrue())
	peUsage, ok := item.(types.PatchEnvelopeUsage)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(peUsage).To(gomega.BeEquivalentTo(expected))
}

func TestHandleAppInstanceDiscovery(t *testing.T) {
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	logger := logrus.StandardLogger()
	log := base.NewSourceLogObject(logger, "pubsub", 1234)
	ps := pubsub.New(pubsub.NewMemoryDriver(), logger, log)

	u, err := uuid.FromString("6ba7b810-9dad-11d1-80b4-000000000000")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	u1, err := uuid.FromString("6ba7b810-9dad-11d1-80b4-000000000001")
	g.Expect(err).ToNot(gomega.HaveOccurred())

	appInstanceStatus, err := ps.NewPublication(pubsub.PublicationOptions{
		AgentName:  "zedmanager",
		TopicType:  types.AppInstanceStatus{},
		Persistent: true,
	})

	a := types.AppInstanceStatus{
		UUIDandVersion: types.UUIDandVersion{
			UUID:    u,
			Version: "1.0",
		},
		AppNetAdapters: []types.AppNetAdapterStatus{
			{
				AllocatedIPv4Addr: net.ParseIP("192.168.1.1"),
				AppNetAdapterConfig: types.AppNetAdapterConfig{
					IfIdx:           2,
					AllowToDiscover: true,
				},
			},
		},
	}
	err = appInstanceStatus.Publish(u.String(), a)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	discoverableNet := types.AppNetAdapterStatus{
		AllocatedIPv4Addr: net.ParseIP("192.168.1.2"),
		VifInfo:           types.VifInfo{VifConfig: types.VifConfig{Vif: "eth0"}},
	}
	a1 := types.AppInstanceStatus{
		UUIDandVersion: types.UUIDandVersion{
			UUID:    u1,
			Version: "1.0",
		},
		AppNetAdapters: []types.AppNetAdapterStatus{discoverableNet},
	}
	err = appInstanceStatus.Publish(u1.String(), a1)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	srv := &msrv.Msrv{
		Log:    log,
		PubSub: ps,
		Logger: logger,
	}

	dir, err := ioutil.TempDir("/tmp", "msrv_test")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer os.RemoveAll(dir)

	err = srv.Init(dir, true)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	err = srv.Activate()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	handler := srv.MakeMetadataHandler()

	descReq := httptest.NewRequest(http.MethodGet, "/eve/v1/discover-network.json", nil)
	descReq.RemoteAddr = "192.168.1.1:0"
	descResp := httptest.NewRecorder()

	handler.ServeHTTP(descResp, descReq)
	g.Expect(descResp.Code).To(gomega.Equal(http.StatusOK))

	defer descResp.Body.Reset()
	var got map[string][]msrv.AppInstDiscovery

	err = json.NewDecoder(descResp.Body).Decode(&got)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	expected := map[string][]msrv.AppInstDiscovery{
		u1.String(): {{
			Port:    discoverableNet.Vif,
			Address: discoverableNet.AllocatedIPv4Addr.String(),
		}},
	}
	g.Expect(got).To(gomega.BeEquivalentTo(expected))
}

func TestTpmActivateCred(t *testing.T) {
	// FIX-ME: to the SWTPM dance here
	// This test conatains TPM kong-fu, not for the faint of heart.
	t.Parallel()
	g := gomega.NewGomegaWithT(t)

	logger := logrus.StandardLogger()
	log := base.NewSourceLogObject(logger, "pubsub", 1234)
	ps := pubsub.New(pubsub.NewMemoryDriver(), logger, log)

	srv := &msrv.Msrv{
		Log:    log,
		PubSub: ps,
		Logger: logger,
	}

	dir, err := os.MkdirTemp("/tmp", "msrv_test")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	defer os.RemoveAll(dir)

	err = srv.Init(dir, true)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	err = srv.Activate()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	handler := srv.MakeMetadataHandler()

	// Get the activate credential parameters
	pCred := httptest.NewRequest(http.MethodGet, "/eve/v1/tpm/activatecredential/", nil)
	pCred.RemoteAddr = "192.168.1.1:0"
	pCredRec := httptest.NewRecorder()

	handler.ServeHTTP(pCredRec, pCred)
	defer pCredRec.Body.Reset()
	g.Expect(pCredRec.Code).To(gomega.Equal(http.StatusOK))

	var credParam msrv.ActivateCredTpmParam
	err = json.Unmarshal(pCredRec.Body.Bytes(), &credParam)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Decode the EK back to tpm2.Public, in practices you need to find a
	// way to trust EK using decive cert or OEM cert or whatever.
	eKBytes, err := base64.StdEncoding.DecodeString(credParam.Ek)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	ekPub, err := tpm2.DecodePublic(eKBytes)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Decode the name back to name tpm2.Name
	nameBytes, err := base64.StdEncoding.DecodeString(credParam.AikName)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	name, err := tpm2.DecodeName(bytes.NewBuffer(nameBytes))
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Decode the AIK back to tpm2.Public
	aikBytes, err := base64.StdEncoding.DecodeString(credParam.AikPub)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	aikPub, err := tpm2.DecodePublic(aikBytes)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Verify the name matches the AIK
	h, err := name.Digest.Alg.Hash()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	p, err := aikPub.Encode()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	aikPubHash := h.New()
	aikPubHash.Write(p)
	aikPubDigest := aikPubHash.Sum(nil)
	g.Expect(bytes.Equal(name.Digest.Value, aikPubDigest)).To(gomega.BeTrue())

	// Verify the AIK is a restricted signing key
	g.Expect((aikPub.Attributes & tpm2.FlagFixedTPM)).To(gomega.BeEquivalentTo(tpm2.FlagFixedTPM))
	g.Expect((aikPub.Attributes & tpm2.FlagRestricted)).To(gomega.BeEquivalentTo(tpm2.FlagRestricted))
	g.Expect((aikPub.Attributes & tpm2.FlagFixedParent)).To(gomega.BeEquivalentTo(tpm2.FlagFixedParent))
	g.Expect((aikPub.Attributes & tpm2.FlagSensitiveDataOrigin)).To(gomega.BeEquivalentTo(tpm2.FlagSensitiveDataOrigin))

	// Generate a credential
	encKey, err := ekPub.Key()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	credential := []byte{0x5A, 0x45, 0x44, 0x45, 0x44, 0x41}
	symBlockSize := int(ekPub.RSAParameters.Symmetric.KeyBits) / 8
	credBlob, encryptedSecret, err := cred.Generate(name.Digest, encKey, symBlockSize, credential)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	var activeCredParam msrv.ActivateCredGenerated
	activeCredParam.Cred = base64.StdEncoding.EncodeToString(credBlob)
	activeCredParam.Secret = base64.StdEncoding.EncodeToString(encryptedSecret)
	jsonStr, err := json.Marshal(activeCredParam)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Do the activation dance
	aCred := httptest.NewRequest(http.MethodPost, "/eve/v1/tpm/activatecredential/", bytes.NewBuffer(jsonStr))
	aCred.RemoteAddr = "192.168.1.1:0"
	aCredRec := httptest.NewRecorder()

	handler.ServeHTTP(aCredRec, aCred)
	defer aCredRec.Body.Reset()
	g.Expect(aCredRec.Code).To(gomega.Equal(http.StatusOK))

	var actCred msrv.ActivateCredActivated
	err = json.Unmarshal(aCredRec.Body.Bytes(), &actCred)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	recovered, err := base64.StdEncoding.DecodeString(actCred.Secret)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(bytes.Equal(recovered, credential)).To(gomega.BeTrue())

	// Verify the EK signature
	digest, err := base64.StdEncoding.DecodeString(actCred.Digest)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	sig, err := base64.StdEncoding.DecodeString(actCred.Sig)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	sinerPubKey, err := aikPub.Key()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	sinerPub := sinerPubKey.(*rsa.PublicKey)
	err = rsa.VerifyPKCS1v15(sinerPub, crypto.SHA256, digest[:], sig)
	g.Expect(err).ToNot(gomega.HaveOccurred())
}
