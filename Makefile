# Change these variables as necessary.
main_package_path = ./cmd/bender
binary_name = bender

# The CLIs live in their own module so the app doesn't inherit their
# dependency trees. -modfile runs them from that module in place.
tools = -modfile=tools/go.mod

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)"
	go vet ./...
	GOOS=windows go vet ./...
	GOOS=linux go vet ./...
	go tool $(tools) govulncheck ./...
	GOOS=windows go build -o /dev/null $(main_package_path)
	GOOS=linux go build -o /dev/null $(main_package_path)
	cd tools && go mod tidy -diff
	cd tools && go mod verify

## test: run all tests
.PHONY: test
test:
	go test -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	cd tools && go mod tidy -v
	go fmt ./...

## generate: regenerate templ and sqlc code and Windows resources
.PHONY: generate
generate:
	go tool $(tools) templ generate -path internal/chrome
	go tool $(tools) sqlc generate
	go tool $(tools) go-winres make --in cmd/bender/winres/winres.json --out cmd/bender/rsrc

## ui: rebuild the embedded chrome stylesheet (loom entry file + Tailwind)
.PHONY: ui
ui: generate
	go run github.com/pietjan/loom/cmd/css -o internal/chrome/css/loom.css
	tailwindcss -i internal/chrome/css/input.css -o internal/chrome/styles.css --minify

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

.PHONY: release/version
release/version:
	@echo '$(version)' | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { echo 'usage: make release version=vX.Y.Z'; exit 1; }

## release: tag and push a release, e.g. make release version=v0.1.0
.PHONY: release
release: release/version no-dirty audit confirm
	git tag -a $(version) -m '$(version)'
	git push origin $(version)

## build: build the Windows binary into bin/
.PHONY: build
build: ui
	GOOS=windows GOARCH=amd64 go build -ldflags='-s -w -H=windowsgui' -o bin/$(binary_name).exe $(main_package_path)

## build/debug: build with a console window attached (log output visible)
.PHONY: build/debug
build/debug: ui
	GOOS=windows GOARCH=amd64 go build -o bin/$(binary_name)-debug.exe $(main_package_path)

# Launching a fresh exe straight off \\wsl$ hangs before main() (Defender
# scans the unsigned binary over the network path, sometimes forever), so
# run copies it to the Windows temp dir first.
win_temp = $(shell wslpath "$$(/mnt/c/Windows/System32/cmd.exe /c 'echo %TEMP%' 2>/dev/null | tr -d '\r')")

## run: build and launch the app (works from WSL2 via Windows interop)
.PHONY: run
run: build
	cp bin/$(binary_name).exe "$(win_temp)/$(binary_name).exe"
	"$(win_temp)/$(binary_name).exe"

## run/debug: launch a console build with DevTools enabled
.PHONY: run/debug
run/debug: build/debug
	cp bin/$(binary_name)-debug.exe "$(win_temp)/$(binary_name)-debug.exe"
	"$(win_temp)/$(binary_name)-debug.exe" -debug

# WSLg env for launching from a bare (non-login) shell.
wslg_env = DISPLAY=:0 WAYLAND_DISPLAY=wayland-0 XDG_RUNTIME_DIR=/mnt/wslg/runtime-dir

## build/linux: build the Linux binary into bin/
.PHONY: build/linux
build/linux: ui
	GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o bin/$(binary_name) $(main_package_path)

## run/linux: build and launch the Linux build (WSLg-aware) with DevTools
.PHONY: run/linux
run/linux: build/linux
	$(wslg_env) ./bin/$(binary_name) -debug
