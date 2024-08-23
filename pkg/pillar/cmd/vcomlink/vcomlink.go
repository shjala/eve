// Copyright (c) 2018-2024 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package vcomlink

import (
	"encoding/json"
	"flag"
	"time"

	"github.com/lf-edge/eve/pkg/pillar/agentbase"
	"github.com/lf-edge/eve/pkg/pillar/agentlog"
	"github.com/lf-edge/eve/pkg/pillar/base"
	"github.com/lf-edge/eve/pkg/pillar/pubsub"
	"github.com/lf-edge/eve/pkg/pillar/types"
	"github.com/lf-edge/eve/pkg/pillar/vcom"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	agentName     = "vcomlink"
	errorTime     = 3 * time.Minute
	warningTime   = 40 * time.Second
	maxPacketSize = 4096
	backLogSize   = unix.SOMAXCONN
)

var (
	logger *logrus.Logger
	log    *base.LogObject
)

type vcomLinkContext struct {
	agentbase.AgentBase
	ps              *pubsub.PubSub
	subGlobalConfig pubsub.Subscription

	GCInitialized bool
	// cli options
}

// AddAgentSpecificCLIFlags adds CLI options
func (ctx *vcomLinkContext) AddAgentSpecificCLIFlags(flagSet *flag.FlagSet) {
}

// Run is the entry point for vcomlink, from zedbox
//
//nolint:funlen,gocognit,gocyclo
func Run(ps *pubsub.PubSub, loggerArg *logrus.Logger, logArg *base.LogObject, arguments []string, baseDir string) int {
	logger = loggerArg
	log = logArg

	ctx := vcomLinkContext{
		ps: ps,
	}
	agentbase.Init(&ctx, logger, log, agentName,
		agentbase.WithPidFile(),
		agentbase.WithBaseDir(baseDir),
		agentbase.WithWatchdog(ps, warningTime, errorTime),
		agentbase.WithArguments(arguments))

	// Run a periodic timer so we always update StillRunning
	stillRunning := time.NewTicker(25 * time.Second)
	ps.StillRunning(agentName, warningTime, errorTime)

	// Look for global config such as log levels
	subGlobalConfig, err := ps.NewSubscription(pubsub.SubscriptionOptions{
		AgentName:     "zedagent",
		MyAgentName:   agentName,
		TopicImpl:     types.ConfigItemValueMap{},
		Persistent:    true,
		Activate:      false,
		Ctx:           &ctx,
		CreateHandler: handleGlobalConfigCreate,
		ModifyHandler: handleGlobalConfigModify,
		DeleteHandler: handleGlobalConfigDelete,
		WarningTime:   warningTime,
		ErrorTime:     errorTime,
	})
	if err != nil {
		log.Fatal(err)
	}
	ctx.subGlobalConfig = subGlobalConfig
	subGlobalConfig.Activate()

	for !ctx.GCInitialized {
		log.Functionf("waiting for GCInitialized")
		select {
		case change := <-subGlobalConfig.MsgChan():
			subGlobalConfig.ProcessChange(change)
		case <-stillRunning.C:
		}
		ps.StillRunning(agentName, warningTime, errorTime)
	}
	log.Functionf("processed GlobalConfig")

	// start listening on vsock
	go startVsockServer()

	for {
		select {
		case change := <-subGlobalConfig.MsgChan():
			subGlobalConfig.ProcessChange(change)
		case <-stillRunning.C:
		}
		ps.StillRunning(agentName, warningTime, errorTime)
	}
}

func handleGlobalConfigCreate(ctxArg interface{}, key string,
	statusArg interface{}) {
	handleGlobalConfigImpl(ctxArg, key, statusArg)
}

func handleGlobalConfigModify(ctxArg interface{}, key string,
	statusArg interface{}, oldStatusArg interface{}) {
	handleGlobalConfigImpl(ctxArg, key, statusArg)
}

func handleGlobalConfigImpl(ctxArg interface{}, key string,
	statusArg interface{}) {

	ctx := ctxArg.(*vcomLinkContext)
	if key != "global" {
		log.Functionf("handleGlobalConfigImpl: ignoring %s", key)
		return
	}
	log.Functionf("handleGlobalConfigImpl for %s", key)
	gcp := agentlog.HandleGlobalConfig(log, ctx.subGlobalConfig, agentName,
		ctx.CLIParams().DebugOverride, logger)
	if gcp != nil {
		ctx.GCInitialized = true
	}
	log.Functionf("handleGlobalConfigImpl done for %s", key)
}

func handleGlobalConfigDelete(ctxArg interface{}, key string,
	statusArg interface{}) {

	ctx := ctxArg.(*vcomLinkContext)
	if key != "global" {
		log.Functionf("handleGlobalConfigDelete: ignoring %s", key)
		return
	}
	log.Functionf("handleGlobalConfigDelete for %s", key)
	agentlog.HandleGlobalConfig(log, ctx.subGlobalConfig, agentName,
		ctx.CLIParams().DebugOverride, logger)
	log.Functionf("handleGlobalConfigDelete done for %s", key)
}

func startVsockServer() {
	// XXX : this rudimentary vsock server, it still can handle multiple VMs
	// but if it gets too complex in the future it can be improved by
	// assigning each vm or service a unique port.
	sock, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Errorf("failed to create vsock socket: %v", err)
		return
	}
	defer unix.Close(sock)

	addr := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_HOST,
		Port: vcom.HostVPort,
	}
	if err := unix.Bind(sock, addr); err != nil {
		log.Errorf("failed to bind vsock socket: %v", err)
		return
	}
	if err := unix.Listen(sock, backLogSize); err != nil {
		log.Errorf("failed to listen on vsock socket: %v", err)
		return
	}
	log.Noticef("Listening on vsock CID %d, port %d", addr.CID, addr.Port)

	for {
		conn, _, err := unix.Accept(sock)
		if err != nil {
			log.Errorf("failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(fd int) {
	defer unix.Close(fd)

	buffer := make([]byte, maxPacketSize)
	n, err := unix.Read(fd, buffer)
	if err != nil {
		log.Errorf("Error reading from connection %v", err)
		return
	}
	buffer = buffer[:n]

	packet := vcom.BasePacket{}
	err = json.Unmarshal(buffer, &packet)
	if err != nil {
		log.Errorf("error unmarshalling packet %v", err)
		respondWithDefaultError(fd)
		return
	}

	switch packet.Channel {
	case vcom.ChannelTpm:
		res, err := handleTPMCom(buffer)
		if err != nil {
			log.Errorf("error handling tpm packet %v", err)
			respondWithDefaultError(fd)
			return
		}
		writeResponse(fd, res)
	default:
		respondWithDefaultError(fd)
	}
}

func respondWithDefaultError(fd int) {
	jsonBytes, _ := json.Marshal(vcom.ErrorResponse{
		Error: "received malformed packet",
	})
	_, _ = unix.Write(fd, jsonBytes)
}

func writeResponse(fd int, response []byte) {
	_, err := unix.Write(fd, response)
	if err != nil {
		log.Errorf("Error writing response to connection %v", err)
	}
}
