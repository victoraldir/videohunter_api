STACK_NAME ?= videohunter-api
REGION := us-east-1
APP_FOLDER := videohunter-api
APP_LOCAL_NETWORK := myvideohunter-api
FUNCTIONS := create-url get-url download-video-hls mix-audio-video create-url-batch
# Get token from env variable
PARAMETERS_OVERRIDE := LogLevel=INFO

# To try different version of Go
GO := go

# Make sure to install aarch64 GCC compilers if you want to compile with GCC.
CC := aarch64-linux-gnu-gcc
GCCGO := aarch64-linux-gnu-gccgo-10

.PHONY: build

build: build-sam
	${MAKE} ${MAKEOPTS} $(foreach function,${FUNCTIONS}, build-${function})

build-%:
	cd ${APP_FOLDER}/functions/$* && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 ${GO} build -o bootstrap

build-sam: build build/layer/bin/ffmpeg
	@sam build

test:
	@cd ${APP_FOLDER} && go test -tags=unit -race -coverprofile=../coverage.txt -covermode=atomic -timeout 5s ./...

.PHONY: tidy
tidy:
	@$(foreach dir,$(MODULE_DIRS),(cd $(dir) && go mod tidy) &&) true

clean:
	@rm -f $(foreach function,${FUNCTIONS}, ${APP_FOLDER}/functions/${function}/bootstrap)
	@rm -rf build

build/layer/bin/ffmpeg:
	mkdir -p build/bin
	cd build && curl https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-5.1.1-arm64-static.tar.xz | tar xJ
	mv build/ffmpeg*/ffmpeg build/ffmpeg*/ffprobe build/bin
	chmod +x build/bin/ffmpeg build/bin/ffprobe
	cd build && zip -r9 layer.zip bin
	rm -rf build/ffmpeg*
	rm -rf build/bin
	

delete:
	@sam delete --stack-name ${STACK_NAME} --region ${REGION} 

deploy: build-sam
	if [ -f samconfig.toml ]; \
		then sam deploy --stack-name ${STACK_NAME} --region ${REGION} --parameter-overrides ${PARAMETERS_OVERRIDE} --no-confirm-changeset; \
		else sam deploy -g --stack-name ${STACK_NAME} --region ${REGION} --parameter-overrides ${PARAMETERS_OVERRIDE} --no-confirm-changeset; \
  	fi

list-resources:
	@sam list endpoints --stack-name ${STACK_NAME} --region ${REGION}

run-local: build-sam
	cd ${APP_FOLDER} && docker compose up -d
	@sam local start-api --docker-network ${APP_LOCAL_NETWORK} -n environments/local.json

run-local-lambda: build-sam
	cd ${APP_FOLDER} && docker compose up -d
	@sam build && sam local start-lambda --docker-network ${APP_LOCAL_NETWORK} -n environments/local.json

export GOBIN ?= $(shell pwd)/bin

STATICCHECK = $(GOBIN)/staticcheck

# Many Go tools take file globs or directories as arguments instead of packages
GO_FILES := $(shell \
	       find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	       -o -name '*.go' -print | cut -b3-)

.PHONY: lint
lint: install-staticcheck
	@staticcheck ./...

.PHONY: install-staticcheck
install-staticcheck:
	@go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: lint
lint: $(STATICCHECK)
	@rm -rf lint.log
	@echo "Checking formatting..."
	@gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	@echo "Checking vet..."
	@$(foreach dir,$(APP_FOLDER),(cd $(dir) && go vet ./... 2>&1) &&) true | tee -a lint.log
	@echo "Checking staticcheck..."
	@$(foreach dir,$(APP_FOLDER),(cd $(dir) && $(STATICCHECK) ./... 2>&1) &&) true | tee -a lint.log
	@echo "Checking for unresolved FIXMEs..."
	@git grep -i fixme | grep -v -e Makefile | tee -a lint.log
	@[ ! -s lint.log ]
	@rm lint.log
	@echo "Checking 'go mod tidy'..."
	@make tidy
	@if ! git diff --quiet; then \
		echo "'go diff tidy' resulted in chnges or working tree is dirty:"; \
		git --no-pager diff; \
	fi

$(STATICCHECK):
	cd tools && go install honnef.co/go/tools/cmd/staticcheck