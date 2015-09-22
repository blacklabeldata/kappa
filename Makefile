proj = kappa
binary = $(proj)
datadir = data
CWD=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

export LOGXI=*=DBG
export LOGXI_COLORS=key=green,value=magenta,message=cyan,TRC,DBG,WRN=red+h,INF=green,ERR=red+h,maxcol=1000
export LOGXI_FORMAT=happy,t="2006-01-02 15:04:05.000000"
export GIN_MODE=release
# export GODEP=$GOPATH/bin/godep

default: build
build: **/*.go
	@echo "------------------"
	@echo " Building binary"
	@echo "------------------"
	@godep go build .

test: **/*.go
	@echo "------------------"
	@echo " test"
	@echo "------------------"
	@godep go test ./...

deps:
	@echo "------------------"
	@echo " Downloading deps"
	@echo "------------------"
	@godep get .

clean:
	@echo "------------------"
	@echo " Cleaning"
	@echo "------------------"
	@rm $(binary)

ca: build
	./$(binary) init-ca

cert: build
	./$(binary) new-cert

setup: build
	./$(binary) init-ca
	./$(binary) new-cert
	./$(binary) new-cert --name=admin

run: build
	@mkdir -p $(datadir)
	./$(binary) server --http-listen=:19022 --ssh-listen=:9022 -D=data --ssh-key=pki/private/localhost.key --ca-cert=pki/ca.crt --admin-cert=pki/public/admin.crt

docker: export GOOS=linux
docker: export CGO_ENABLED=0
docker: export GOARCH=amd64
docker:
	@$(GOPATH)/bin/godep go build -a -installsuffix cgo -o kappa .
	@docker build -t kappa -f Dockerfile.scratch .

html:
	@echo "------------------"
	@echo " html report"
	@echo "------------------"
	@go tool cover -html=$(CWD)/coverage.out -o $(CWD)/coverage.html
	@open coverage.html

detail:
	@echo "------------------"
	@echo " detailed report"
	@echo "------------------"
	@gocov test ./... | gocov report

report: test detail html


.PHONY: build deps clean run
