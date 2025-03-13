BIN           := `grep "module" ./go.mod | sed 's/module //'`
REVISION      := `git rev-parse --short HEAD`
FLAG          :=  -a -tags netgo -trimpath -ldflags='-s -w -extldflags="-static" -buildid='
RESOURCE_DIR  := resources
BINDATA_FILE  := bindata.go
SOURCE_FILES  := go.mod go.sum *.go makefile .gitignore readme.md bash.exe

.PHONY: all build clean release update-binary

all:
	cat ./makefile | grep '^[^ ]*:$$'
build:
	make update-binary
	go build -o $(BIN).exe

release:
	make clean-bindata
	make build
	GOOS=windows go build $(FLAG) -o $(BIN).exe
	$(RESOURCE_DIR)/upx --lzma $(BIN).exe
	echo Success!

rebuild:
	make clean
	make release

# 埋め込むデータの更新
update-binary:
	if ! [ -e "$(RESOURCE_DIR)" ]; then \
		mkdir -p $(RESOURCE_DIR); \
		mkdir -p $(RESOURCE_DIR)/src; \
		make get-busybox; \
		make get-upx; \
		make get-jq; \
		make get-nkf; \
	fi
	cp -rfp $(SOURCE_FILES) $(RESOURCE_DIR)/src/; \
	rm -rf $(RESOURCE_DIR)/src/bindata.go; \
	make gen-bindata

gen-bindata:
	if which go-bindata >/dev/null; then :; else go install github.com/go-bindata/go-bindata/...@latest ; fi
	go-bindata -o $(BINDATA_FILE) $(RESOURCE_DIR)/ $(RESOURCE_DIR)/src/

clean-bindata:
	rm -rf "$(BINDATA_FILE)"
clean-resource:
	rm -rf "$(RESOURCE_DIR)"
clean:
	make clean-bindata
	make clean-resource
	rm -rf "$(BIN).exe"

get-golang:
	# これは流石にwingetしていいよね...
	winget install GoLang.Go
get-upx:
	# winget install upxでもよい。
	# ここではビルド時にバイナリに埋め込むことを想定して配置する。
	curl -L `curl -s https://api.github.com/repos/upx/upx/releases/latest | grep "browser_download_url" | grep "win64.zip" | cut -d"\"" -f4` -o upx.zip; unzip -jo upx.zip "upx*/upx.exe" -d .; mv upx.exe "$(RESOURCE_DIR)/" ; rm upx.zip
get-busybox:
	curl.exe -L "https://frippery.org/files/busybox/busybox64u.exe" -o "$(RESOURCE_DIR)/busybox64u.exe"
	cp -rfp "$(RESOURCE_DIR)/busybox64u.exe" bash.exe || :
get-jq:
	# winget install jqlang.jqでもよい。
	# ここではビルド時にバイナリに埋め込むことを想定して配置する。
	# jqを使わず、jqの最新版をGitHubから取ってくるのは難しい。1.7.1を取ってきて取る。
	curl.exe -s -L https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-windows-amd64.exe -o "./$(RESOURCE_DIR)/jq171.exe" \
		&& curl -L "$$( \
			curl.exe -s "https://api.github.com/repos/jqlang/jq/releases/latest" \
			| ./$(RESOURCE_DIR)/jq171.exe -r '.assets[].browser_download_url | select(endswith("amd64.exe"))' \
		)" -o $(RESOURCE_DIR)/jq.exe \
		&& rm ./$(RESOURCE_DIR)/jq171.exe \
		&& ./$(RESOURCE_DIR)/jq.exe --version
get-nkf:
	# winget にない。GitHubから最新版を取得する。jqを使用する。とはいえ最新といっても...
	# ここではビルド時にバイナリに埋め込むことを想定して配置する。
	JQ=$$(which jq); [ -n "$$JQ" ] && echo 1 || JQ="./$(RESOURCE_DIR)/jq.exe"; \
		echo "$$JQ"; \
		curl.exe -s "https://api.github.com/repos/kkato233/nkf/releases/latest" \
		| "$$JQ" -r '.assets[].browser_download_url | select(endswith(".zip"))' \
		| { \
				read url; \
				curl.exe -L "$$url" -o "nkf.zip" \
				&& e=$$( unzip -l "nkf.zip" | grep "nkf.exe" | awk '{print $$4}' ); \
				unzip -p "nkf.zip" "$$e" > "./$(RESOURCE_DIR)/nkf.exe" && rm "nkf.zip"; \
			} \
		&& [ -e "./$(RESOURCE_DIR)/nkf.exe" ] \
			&& "./$(RESOURCE_DIR)/nkf.exe" "--version" | head -n1 | cut -c 1-34 \
			|| { \
				echo "Error: Download nkf failed !!!" >&2; \
				exit 1; \
			}

