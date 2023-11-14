FLAGS := -ldflags "-s -w" -trimpath
NOCGO := CGO_ENABLED=0

ci:: build-all
	echo "done"

build::
	go vet && go fmt
	${NOCGO} go build ${FLAGS} -o st

build-all:: build
	${NOCGO}  GOOS=linux    GOARCH=amd64  go build ${FLAGS} -o builds/st-linux-x64
	${NOCGO}  GOOS=linux    GOARCH=arm    go build ${FLAGS} -o builds/st-linux-arm
	${NOCGO}  GOOS=linux    GOARCH=arm64  go build ${FLAGS} -o builds/st-linux-arm64
	${NOCGO}  GOOS=darwin   GOARCH=amd64  go build ${FLAGS} -o builds/st-mac-x64
	${NOCGO}  GOOS=darwin   GOARCH=arm64  go build ${FLAGS} -o builds/st-mac-arm64
	${NOCGO}  GOOS=windows  GOARCH=amd64  go build ${FLAGS} -o builds/st-windows.exe
	sha256sum builds/*

clean::
	rm -f st
	rm -f st-linux64
	rm -f st-linux-arm
	rm -f st-linux-arm64
	rm -f st-mac
	rm -f st-windows.exe
