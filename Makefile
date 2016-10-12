all:
	go install github.com/ssddanbrown/haste
run:
	go install github.com/ssddanbrown/haste && haste testfile.html
build:
	mkdir builds
	env GOOS=windows GOARCH=386 go build -o builds/haste-windows-386.exe
	env GOOS=linux GOARCH=amd64 go build -o builds/haste-linux-amd64
	env GOOS=darwin GOARCH=amd64 go build -o builds/haste-darwin-amd64