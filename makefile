MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
PARENT_DIR   := $(shell dirname ${MAKEFILE_DIR})

BIN_NAME     := xml2json
DST          := $(MAKEFILE_DIR)
SRC          := $(DST)
BIN          := $(SRC)/$(BIN_NAME)
GOARCH       := amd64
VERSION      := 0.1

FLAGS_VERSION  := -X main.version=$(VERSION) -X main.revision=$(git rev-parse --short HEAD)
FLAGS          := -installsuffix netgo -trimpath "-ldflags=-buildid=" -ldflags '-s -w -extldflags "-static"'
FLAGS_WIN      := -installsuffix netgo -trimpath "-ldflags=-buildid=" -ldflags '-s -w -extldflags "-static" -H windowsgui'
#FLAGS_WIN      := -tags netgo -installsuffix netgo -trimpath "-ldflags=-buildid=" -ldflags '-s -w -extldflags "-static"'

BUILD_TAG_RELEASE := -tags netgo
BUILD_TAG         := -tags debug $(BUILD_TAG_RELEASE)

all:
	# make win
	# make linux
	# make mac
	# make pi
	# make info

release:
	make clean
	cd $(SRC) && GOARCH=amd64 GOOS=windows go build -o $(DST)/$(BIN_NAME).exe $(BUILD_TAG_RELEASE) $(FLAGS_WIN)
	upx $(DST)/$(BIN_NAME).exe
	mv $(DST)/$(BIN_NAME).exe $(DST)/$(BIN_NAME)_windows_amd64.exe
	cd $(SRC) && GOARCH=$(GOARCH) GOOS=linux go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG_RELEASE) $(FLAGS)
	upx $(DST)/$(BIN_NAME)
	mv $(DST)/$(BIN_NAME) $(DST)/$(BIN_NAME)_linux_amd64
	cd $(SRC) && GOARCH=amd64 GOOS=darwin go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG_RELEASE) $(FLAGS)
	mv $(DST)/$(BIN_NAME) $(DST)/$(BIN_NAME)_darwin_amd64
	cd $(SRC) && GOARM=6  GOARCH=arm  GOOS=linux go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG_RELEASE) $(FLAGS)
	mv $(DST)/$(BIN_NAME) $(DST)/$(BIN_NAME)_linux_armv6l

info:
	@echo "============================================================================"
	@echo "MAKEFILE_DIR : $(MAKEFILE_DIR)"
	@echo "DST          : $(DST)"
	@echo "SRC          : $(SRC)"
	@echo "BIN          : $(BIN)"
	@echo "BIN_NAME     : $(BIN_NAME)"
	@echo "============================================================================"

clean:
	rm -rf $(DST)/$(BIN_NAME) $(DST)/$(BIN_NAME).exe $(DST)/$(BIN_NAME)_win.exe $(DST)/$(BIN_NAME)_linux $(DST)/$(BIN_NAME)_darwin $(DST)/$(BIN_NAME)_armv6l

win:
	if [ -e $(DST)/$(BIN_NAME).exe ]; then rm -rf $(DST)/$(BIN_NAME).exe; fi
	#GOARCH=$(GOARCH) GOOS=windows go build -o $(DST)/$(BIN)_windows.exe $(FLAGS_WIN) 
	cd $(SRC) && GOARCH=amd64 GOOS=windows go build -o $(DST)/$(BIN_NAME).exe $(BUILD_TAG) $(FLAGS_WIN)
	upx $(DST)/$(BIN_NAME).exe

linux:
	if [ -e $(DST)/$(BIN_NAME) ]; then rm -rf $(DST)/$(BIN_NAME); fi
	cd $(SRC) && GOARCH=$(GOARCH) GOOS=linux go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG) $(FLAGS)
	upx $(DST)/$(BIN_NAME)

mac:
	if [ -e $(DST)/$(BIN_NAME) ]; then rm -rf $(DST)/$(BIN_NAME); fi
	cd $(SRC) && GOARCH=amd64 GOOS=darwin go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG) $(FLAGS)
	# mac用のバイナリをupxで圧縮するとセグフォが多発した
	# upx $(DST)/$(BIN_NAME)

pi:
	if [ -e $(DST)/$(BIN_NAME) ]; then rm -rf $(DST)/$(BIN_NAME); fi
	cd $(SRC) && GOARM=6  GOARCH=arm  GOOS=linux go build -o $(DST)/$(BIN_NAME) $(FLAGS_UNIX) $(BUILD_TAG) $(FLAGS)
	# raspberry pi用ののバイナリをupxで圧縮するとセグフォが多発した
	#rm -rf $(DST)/$(BIN_NAME).upx && upx $(DST)/$(BIN_NAME) -o $(DST)/$(BIN_NAME).upx
	#rm -rf $(DST)/$(BIN_NAME)
	#mv $(DST)/$(BIN_NAME).upx $(DST)/$(BIN_NAME)
	until cp -f $(DST)/$(BIN_NAME) ~/rpi/go/go/bin; do sleep 1; done
	#until cp -rf ../s  ~/rpi/work/; do sleep 1; done

install_upx:
	until sudo apt install upx -y --fix-missing; do sleep 1; done


