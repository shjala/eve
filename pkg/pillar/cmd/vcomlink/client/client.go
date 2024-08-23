package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/lf-edge/eve/pkg/pillar/vcom"
	"golang.org/x/sys/unix"
)

func getTpmEkFromHost(fd int) error {
	request := vcom.TpmRequest{
		BasePacket: vcom.BasePacket{
			Channel: vcom.ChannelTpm,
		},
		Request: vcom.RequestTpmGetEk,
	}

	packetBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %v", err)
	}

	fmt.Printf("Sending packet to server %s\n", string(packetBytes))
	_, err = unix.Write(fd, []byte(packetBytes))
	if err != nil {
		return fmt.Errorf("failed to send packet to server: %v", err)
	}
	fmt.Println("[+] Message sent to server")

	buffer := make([]byte, 4096)
	n, err := unix.Read(fd, buffer)
	if err != nil {
		return fmt.Errorf("failed to read response from server: %v", err)
	}

	fmt.Printf("Received response from server: %s\n", string(buffer[:n]))
	return nil
}

func main() {
	tpmFlag := flag.Bool("tpmek", false, "get TPM Endorsement Key")
	flag.Parse()

	if *tpmFlag {
		fmt.Println("[+] Getting TPM Endorsement Key...")
	} else {
		fmt.Fprintf(os.Stderr, "No function specified\n")
		os.Exit(1)
	}

	// Create a vsock socket
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] failed to create vsock socket: %v\n", err)
		os.Exit(1)
	}
	defer unix.Close(fd)

	// Define the vsock address
	addr := &unix.SockaddrVM{
		CID:  unix.VMADDR_CID_HOST,
		Port: vcom.HostVPort,
	}
	err = unix.Connect(fd, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] failed to connect to vsock server: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[+] Connected to vsock server")

	if *tpmFlag {
		err = getTpmEkFromHost(fd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] failed to communicate with server: %v\n", err)
			os.Exit(1)
		}
	}
}
