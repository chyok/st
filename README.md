# Simple Transfer

![GitHub tag (with filter)](https://img.shields.io/github/v/tag/chyok/st)
![GitHub License](https://img.shields.io/github/license/chyok/st)

`st` is a command-line file transfer tool for local networks. It has a built-in LAN discovery feature, allowing easy file sharing between devices.

![example](https://github.com/chyok/st/assets/32629225/a638b0d2-f509-4e34-a99b-9f9e2a757e02)

## Simple Usage
1. **Receive Files**: - Run `st` to start the file reception service and display a QR code. - Another device can scan the QR code or access the displayed service address to upload files.

2. **Send Files**: - Run `st [filename|foldername]` to start the file sending service and display a QR code. - Another device can scan the QR code or access the displayed service address to download the file.

3. **Automatic discovery**: If both devices have `st` running:

   Device A: `st`  

   Device B: `st xxx.txt`  send file to A  

   ------

   Device A: `st xxx.txt`

   Device B: `st`  receive file from A

## Features  

`st` offers a convenient and quick method for file transfer within a local network.  

- Web-based file transfer interface
- QR code for more convenient transfer between mobile phone and pc.
- Support for transferring both files and folders
- Automatic discovery of hosts within a local network

## Installation 

### Binaries on macOS, Linux, Windows

Download from [Github Releases](https://github.com/chyok/st/releases), add st to your $PATH.

### Build from Source  

```
go install github.com/chyok/st@latest
```

## Command  

`st` 
start a receive server and display a QR code., waiting for upload.

`st [filename|foldername]` 
start a send server and display a QR code., waiting for download.

`st -p [port]` 
manually specify the service port and multicast port, the default is 53333.


## License  

MIT. See [LICENSE](https://github.com/chyok/st/blob/main/LICENSE).  
