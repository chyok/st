# Simple Transfer

`st` is a command-line project written in Go for file transfer within a local network. Its working principle is based on multicast, where the sender discovers the host in the local network, and the receiver returns its IP address to the sender and initiates an HTTP service to receive files. The sender then transfers the file to the receiver via HTTP. Moreover, even if there's only a receiver, it can access the file transfer address via the prompted address, allowing file transfer through a webpage without the need to run a command on the sender's side.  

![example](https://github.com/chyok/st/assets/32629225/3f1b2a19-b84c-4c9a-8264-067e438aa58e)

## Simple Usage
Device A: `st`  
Device B: `st xxx.txt`  transfer file to A

Device A: `st`  (will show `server address: http://192.168.31.16:9999`)  
Device B: open `http://192.168.31.16:9999` and upload file

## Features  

`st` offers a convenient and quick method for file transfer within a local network.  

- Discovers hosts within a local network  
- Transfers files via HTTP  
- Provides a web page access method for file transfer  

## Installation 

### Binaries on macOS, Linux, Windows

Download from [Github Releases](https://github.com/chyok/st/releases), add st to your $PATH.

### Build from Source  

```
go install github.com/chyok/st@latest
```

## Command  

`st` 
start transer server, waiting transfer.

`st [filename]` 
transfer file to all servers on the intranet.

`st -p [port]` 
manually specify the service port, the default is 9999.


## License  

MIT. See [LICENSE](https://github.com/chyok/st/blob/main/LICENSE).  
