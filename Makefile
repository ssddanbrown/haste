all:
	go install github.com/ssddanbrown/haste
run:
	go install github.com/ssddanbrown/haste && haste ./testing
clean:
	rm -rf ./dist
watch:
	go install github.com/ssddanbrown/haste && haste -w testfile.html
build:
	rm -rf builds
	mkdir builds
	env GOOS=windows GOARCH=386 go build -o builds/haste.exe
	env GOOS=linux GOARCH=amd64 go build -o builds/haste-linux
	env GOOS=darwin GOARCH=amd64 go build -o builds/haste-osx
