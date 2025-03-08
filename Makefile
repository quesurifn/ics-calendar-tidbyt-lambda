.PHONY: build clean deploy

build:
	env GOARCH=arm64 GOOS=linux go build -ldflags="-s -w" -o bin/bootstrap ics/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock
	
zip:
	zip -j bin/hello.zip bin/bootstrap

deploy: build
	sls deploy --verbose
