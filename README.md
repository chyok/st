# Simple Transfer

![GitHub tag (with filter)](https://img.shields.io/github/v/tag/chyok/st)
![GitHub License](https://img.shields.io/github/license/chyok/st)

`st` is a command-line file transfer tool for local networks. It has a built-in LAN discovery feature, allowing easy file sharing between devices.

![example](https://github.com/chyok/st/assets/32629225/a638b0d2-f509-4e34-a99b-9f9e2a757e02)

## Simple Usage

1. **Receive files**: run `st` to start a receive server and show a QR code. Another device can scan the QR code, or open the address shown, to upload files and folders from its browser.

2. **Send files**: run `st [filename|foldername]` to start a send server and show a QR code. Another device can scan the QR code, or open the address shown, to download them.

3. **Automatic discovery**: if both devices run `st`, they find each other on the LAN:

   Device A: `st`

   Device B: `st xxx.txt`

   Device A is asked whether to download `xxx.txt` from B, and saves it if you answer `y`. It does not matter which device starts first.

Received files are saved in the current directory. An existing file is never overwritten: a second `a.txt` is saved as `a (1).txt`.

## Features

`st` offers a convenient and quick method for file transfer within a local network.

- Web-based file transfer interface.
- QR code for more convenient transfer between mobile phone and PC.
- Support for transferring both files and folders, keeping the folder structure.
- Automatic discovery of hosts within a local network.
- Downloads are confirmed on the receiving device before they start.
- Only the shared file or folder is served: symbolic links pointing outside it are not followed.
- Shared files are always offered as downloads, never opened as web pages.
- Files are saved only once complete; a transfer stopped with Ctrl+C leaves no partial file behind.

## Installation

### Binaries on macOS, Linux, Windows

Download from [GitHub Releases](https://github.com/chyok/st/releases), add `st` to your `$PATH`.

### Build from Source

```
go install github.com/chyok/st@latest
```

## Command

`st`
start a receive server and display a QR code, waiting for upload.

`st [filename|foldername]`
start a send server and display a QR code, waiting for download.

`st -p [port]`
set the server port, also used for discovery; the default is 53333. Devices must use the same port to find each other.

`st -v`
print the version.

On a device with several network interfaces and no internet connection, `st` cannot tell which network the other device is on, so it shows an address and QR code for each interface; use the one on the same network as the other device.

## Development

Requires Go (see `go.mod`), plus [golangci-lint](https://golangci-lint.run) and [GoReleaser](https://goreleaser.com) for linting and release builds.

```
make build      # build ./st
make test       # run the tests with the race detector
make lint       # run golangci-lint
make snapshot   # cross-compile every release target into dist/
```

### Releasing

Publish a release on GitHub with a tag such as `v0.3.0`. The Release workflow then builds the binaries for every platform with GoReleaser and attaches them to the release, together with `sha256sums.txt`. The release title and notes are left as written.

## License

MIT. See [LICENSE](https://github.com/chyok/st/blob/main/LICENSE).
