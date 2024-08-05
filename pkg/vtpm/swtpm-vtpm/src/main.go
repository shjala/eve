// Copyright (c) 2024 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-tpm/tpm2"
	etpm "github.com/lf-edge/eve/pkg/pillar/evetpm"
	fileutils "github.com/lf-edge/eve/pkg/pillar/utils/file"
)

const (
	swtpmPath = "/usr/bin/swtpm"
	// FIX-ME : use constanst from eve types, in another PR
	// TpmdControlSocket is UDS to aks vtpmd to luanch SWTP instances for VMS
	TpmdControlSocket = "/run/swtpm/tpmlaunchd"
	// SwtpmSocketPath is the prefix for the SWTPM socket
	SwtpmSocketPath = "/run/swtpm/%s.sock"
	// SwtpmPidPath is the prefix for the SWTPM pid file
	SwtpmPidPath = "/run/swtpm/%s.pid"

	stateEncryptionKey = "/run/swtpm/binkey"
	swtpmLogPath       = "/run/swtpm/%s.log"
	swtpmStatePath     = "/persist/swtpm/tpm-state-%s"
	maxInstances       = 10
)

var liveInstances = 0

func makeDirs(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// if path already exist MkdirAll won't check the perms,
	// so sure it has the right permissions by applying it again.
	if err := os.Chmod(dir, 0755); err != nil {
		return fmt.Errorf("failed to set permissions for directory: %w", err)
	}

	return nil
}

func HwtpmIsAvailable() bool {
	_, err := os.Stat(etpm.TpmDevicePath)
	return err == nil
}

func runVirtualTpmInstance(id string) error {
	statePath := fmt.Sprintf(swtpmStatePath, id)
	ekPath := path.Join(statePath, "ek.pub")
	logPath := fmt.Sprintf(swtpmLogPath, id)
	sockPath := fmt.Sprintf(SwtpmSocketPath, id)
	pidPath := fmt.Sprintf(SwtpmPidPath, id)
	swtpmArgs := []string{"socket", "--tpm2",
		"--tpmstate", "dir=" + statePath,
		"--ctrl", "type=unixio,path=" + sockPath + ",terminate",
		// FIX-ME: lower the log level, or get rid of it
		"--log", "file=" + logPath + ",level=20",
		"--pid", "file=" + pidPath,
		"--daemon"}

	if err := makeDirs(statePath); err != nil {
		return fmt.Errorf("failed to create vtpm state directory: %w", err)
	}

	if HwtpmIsAvailable() != true {
		log.Println("TPM is not available, running swtpm without state encryption!")

		cmd := exec.Command(swtpmPath, swtpmArgs...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run swtpm: %w", err)
		}
	} else {
		log.Println("TPM is available, running swtpm with state encryption!")

		key, err := etpm.UnsealDiskKey(etpm.DiskKeySealingPCRs)
		if err != nil {
			return fmt.Errorf("unseal operation failed with err: %w", err)
		}

		if err := os.WriteFile(stateEncryptionKey, key, 0644); err != nil {
			return fmt.Errorf("failed to write key to file: %w", err)
		}

		swtpmArgs = append(swtpmArgs, "--key", "file="+stateEncryptionKey+",format=binary,mode=aes-256-cbc,remove=true")
		cmd := exec.Command(swtpmPath, swtpmArgs...)
		if err := cmd.Run(); err != nil {
			// this shall not fail 🧙🏽‍♂️
			rmErr := os.Remove(stateEncryptionKey)
			if rmErr != nil {
				return fmt.Errorf("failed to run swtpm: %w, failed to remove key file %w", err, rmErr)
			}
			return fmt.Errorf("failed to run swtpm: %w", err)
		}

		// wait a bit for SWTPM to initialize
		time.Sleep(3 * time.Second)

		// if we can't read the EK from SWTPM, it is possibly malfunctioning
		// and of no use, so kill it.
		ek, err := GetVirtualTpmEKPub(sockPath)
		if err != nil {
			ekErr := fmt.Errorf("failed to get EK: %w", err)
			// SWTPM should remove the key after reading it, but just in case
			// make sure key is gone.
			rmErr := os.Remove(stateEncryptionKey)
			if rmErr != nil {
				ekErr = fmt.Errorf("%w, failed to remove key file %w", ekErr, rmErr)
			}

			err = KillVirtualTpmInstance(pidPath)
			if err != nil {
				return fmt.Errorf("%w, %w", ekErr, err)
			}
		}

		// if this fails, something is wrong in the system,
		// so kill the SWTPM instance to prevent further state corruption.
		wrErr := fileutils.WriteRename(ekPath, ek)
		if wrErr != nil {
			err = KillVirtualTpmInstance(pidPath)
			if err != nil {
				return fmt.Errorf("%w, %w", wrErr, err)
			}
			return fmt.Errorf("failed to write EK to file: %w", err)
		}
	}

	return nil
}

func KillVirtualTpmInstance(pidPath string) error {
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("failed to kill potentially malfunctioning SWTPM intance, failed to read pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return fmt.Errorf("failed to kill potentially malfunctioning SWTPM intance, failed to convert pid to int: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to kill potentially malfunctioning SWTPM intance, failed to find process with pid %d: %w", pid, err)
	}

	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill potentially malfunctioning SWTPM intance with pid %s: %w", pid, err)
	}

	return nil
}

func InitializeVirtualTpm(sockPath string) error {
	rw, err := tpm2.OpenTPM(sockPath)
	if err != nil {
		return fmt.Errorf("OpenTPM failed with err: %w", err)
	}
	defer rw.Close()

	if err := etpm.CreateKey(rw, etpm.TpmEKHdl, tpm2.HandleEndorsement, etpm.DefaultEkTemplate, false); err != nil {
		return fmt.Errorf("Error in creating Endorsement key: %w ", err)
	}
	if err := etpm.CreateKey(rw, etpm.TpmSRKHdl, tpm2.HandleOwner, etpm.DefaultSrkTemplate, false); err != nil {
		return fmt.Errorf("Error in creating SRK key: %w ", err)
	}
	if err := etpm.CreateKey(rw, etpm.TpmAIKHdl, tpm2.HandleOwner, etpm.DefaultAikTemplate, false); err != nil {
		return fmt.Errorf("Error in creating Attestation key: %w ", err)
	}
	if err := etpm.CreateKey(rw, etpm.TpmQuoteKeyHdl, tpm2.HandleOwner, etpm.DefaultQuoteKeyTemplate, false); err != nil {
		return fmt.Errorf("Error in creating Quote key: %w ", err)
	}
	if err := etpm.CreateKey(rw, etpm.TpmEcdhKeyHdl, tpm2.HandleOwner, etpm.DefaultEcdhKeyTemplate, false); err != nil {
		return fmt.Errorf("Error in creating ECDH key: %w ", err)
	}

	return nil

}

func main() {
	uds, err := net.Listen("unix", TpmdControlSocket)
	if err != nil {
		log.Fatalf("Failed to create vtpm control socket: %v", err)
	}
	defer uds.Close()

	// make sure we remove the socket file on exit
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		os.Remove(TpmdControlSocket)
		os.Exit(0)
	}()

	for {
		conn, err := uds.Accept()
		if err != nil {
			log.Printf("Failed to accept connection over vtpmd control socket: %v", err)
			continue
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	if liveInstances >= maxInstances {
		log.Printf("Error, max number of Virtual TPM instances reached!")
		return
	}

	id, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Printf("Failed to read SWTPM ID from connection: %v", err)
		return
	}
	id = strings.TrimSpace(id)
	if err := runVirtualTpmInstance(id); err != nil {
		log.Printf("Failed to run SWTPM instance: %v", err)
		return
	}

	liveInstances++
}
