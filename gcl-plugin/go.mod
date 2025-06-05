module github.com/lukasngl/godogen/godogen-lint

go 1.24.2

replace github.com/lukasngl/godogen => ..

require (
	github.com/golangci/plugin-module-register v0.1.2
	github.com/lukasngl/godogen v0.1.0-rc.3
	golang.org/x/tools v0.33.0
)

require github.com/iancoleman/strcase v0.3.0 // indirect
