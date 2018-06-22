all:
	cd server && rice embed-go && cd ..
	go install github.com/ssddanbrown/haste
run:
	cd server && rice embed-go && cd ..
	go install github.com/ssddanbrown/haste && haste ./testing
clean:
	rm -rf ./dist
watch:
	cd server && rice embed-go && cd ..
	go install github.com/ssddanbrown/haste && haste -w testfile.html
build:
	rm -rf builds
	mkdir builds
	cd server && rice embed-go && cd ..
	env GOOS=windows GOARCH=386 go build -o builds/haste-windows-386.exe
	env GOOS=linux GOARCH=amd64 go build -o builds/haste-linux-amd64
	env GOOS=darwin GOARCH=amd64 go build -o builds/haste-osx-amd64
