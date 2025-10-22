module github.com/lukasngl/godogen/godogen-gcl

go 1.25

replace (
	github.com/lukasngl/godogen => ../
	github.com/lukasngl/godogen/godogen-lint => ../godogen-lint
)

require (
	github.com/golangci/plugin-module-register v0.1.2
	github.com/lukasngl/godogen/godogen-lint v0.0.0-00010101000000-000000000000
	golang.org/x/tools v0.38.0
)

require github.com/lukasngl/godogen v0.0.0-00010101000000-000000000000 // indirect
