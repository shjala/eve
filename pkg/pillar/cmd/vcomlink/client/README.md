# VCOM Client

The `client.go` file contains the simple implementation of the client functionality for the VComLink. It is responsible for establishing a connection with the host and sending/receiving data. It must be run on the VM.

## Example

Here's an example of what you expect running the client from the VM:

``` bash
ubuntu@vm01:~/test$ ./main --tpmek
[+] Getting TPM Endorsement Key...
[+] Connected to vsock server
Sending packet to server {"destination":2,"id":2,"Request":1}
[+] Message sent to server
Received response from server: {"ek":"AAEACwADALIAIINxl2dEhLP4GpDMjUal1yT9UtduBlILZPKh2hszFGmqAAYAgABDABAIAAAAAAABAMRQYU1p50RyG+ZiJ96jnRRRbssdVioZSfDDb0r4rI1C0GZ1aRgBBXkVOmKw3tw4B9Roh26h4PatkqZTKKDZvJ1u0kqmByj5cDln92c5vSlRsfhbnKWCW5J/zYv3ZKJBDya0n3YaY6WjruDjjF/o3CKJ4pgo2tTyws+Q/7gZ1klLVrK/Fycn7441y39scFuLOQzaAEuuRB0iB9C8XcW47MmJ38CedNzUdHYNF01MyGgHX+bVDhdlNdX5fP5G4b+smKvGBAUyAh9coVjFp+C5wZPpbL6rlwzzFtIjTTNdYkF8ZH0y5r9TvaBdvxT4sjISmQ/6Qtsk9Ni5b6BbU1nzEds="}
ubuntu@vm01:~/test$
```

