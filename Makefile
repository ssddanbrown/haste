all:
	rice embed-go
	go install github.com/ssddanbrown/haste
run:
	rice embed-go
	go install github.com/ssddanbrown/haste && haste testfile.html
watch:
	rice embed-go
	go install github.com/ssddanbrown/haste && haste -w testfile.html
build:
	mkdir builds
	rice embed-go
	env GOOS=windows GOARCH=386 go build -o builds/haste-windows-386.exe
	env GOOS=linux GOARCH=amd64 go build -o builds/haste-linux-amd64
	env GOOS=darwin GOARCH=amd64 go build -o builds/haste-darwin-amd64