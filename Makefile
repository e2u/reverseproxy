OBJ=reverse
PWD=$(shell pwd)
BUILD_DIR=$(PWD)/objs
SOURCES=*.go


.PHONY: default
default: help


help:                              ## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'



.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

	
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/${OBJ} ${SOURCES}
	

	