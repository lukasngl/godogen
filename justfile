### Global Targets

default:
    just --choose

check: build check-fmt check-gen check-tidy lint test

build: build-godogen build-godogen-language-server build-godogen-lint

test *gotestflags="":
    just _test "." {{ gotestflags }}

lint *gclflags:
    just _lint "." {{ gclflags }}

tidy:
    just _tidy "."

check-tidy:
    ./hack/group.sh "🧹" "checking go mod tidy" ./hack/error-on-diff.sh just tidy

gen:
    just _gen "."

check-gen:
    ./hack/group.sh "⚡" "checking generated code" ./hack/error-on-diff.sh just gen

fmt *treefmtflags: && remove-empty-comments
    treefmt {{ treefmtflags }}

remove-empty-comments:
    ./hack/remove-empty-comments.sh

check-fmt:
    ./hack/group.sh "📝" "checking code formatting" ./hack/error-on-diff.sh just fmt

[parallel]
fix: tidy gen fmt (lint "--fix")

### Tool Builds

build-godogen:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    group.sh "🔨" "building godogen" \
        go build -o godogen .

build-godogen-language-server:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    cd godogen-language-server
    group.sh "🔨" "building godogen-language-server" \
        go build -o ../godogen-language-server .

build-godogen-lint:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    group.sh "🔨" "building godogen-lint" \
        go build -o godogen-lint ./cmd/godogen-lint

### Helpers

_build module:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    cd {{ module }}
    group.sh "📦" "{{ module }} downloading dependencies" \
        go mod download
    group.sh "🔨" "{{ module }} compiling go code" \
        go build ./...

_test module *gotestflags:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    ROOT_DIR="$(pwd)"
    mkdir -p "$ROOT_DIR/coverage"
    cd {{ module }}
    group.sh "📦" "{{ module }} downloading dependencies" \
        go mod download

    GOTESTSUM_FORMAT=gotestdox
    if [ -n "$CI" ]; then
        GOTESTSUM_FORMAT=github-actions
    fi
    gotestsum \
        --format-hide-empty-pkg \
        --format $GOTESTSUM_FORMAT -- \
        -coverpkg=./... \
        -coverprofile="$ROOT_DIR/coverage/coverage.out" -covermode=count \
        {{ gotestflags }} ./...

_lint module *gclflags:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    cd {{ module }}
    group.sh "🔍" "{{ module }} run golangci-lint" \
        golangci-lint run {{ gclflags }}

_gen module:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    cd {{ module }}
    group.sh "📦" "{{ module }} downloading dependencies" \
        go mod download
    group.sh "⚡" "{{ module }} running go generate" \
        go generate ./...

_tidy module:
    #!/usr/bin/env sh
    set -e
    export PATH="$(pwd)/hack/:$PATH"
    cd {{ module }}
    group.sh "🧹" "{{ module }} running go mod tidy" \
        go mod tidy
