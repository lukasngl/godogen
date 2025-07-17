module github.com/lukasngl/godogen/_example

go 1.24.2

replace github.com/lukasngl/godogen => ../

tool github.com/lukasngl/godogen/cmd/godogen

tool github.com/lukasngl/godogen/cmd/godogen-lint

require github.com/cucumber/godog v0.15.0

require (
	github.com/cucumber/gherkin/go/v26 v26.2.0 // indirect
	github.com/cucumber/messages/go/v21 v21.0.1 // indirect
	github.com/gofrs/uuid v4.3.1+incompatible // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/lukasngl/godogen v0.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
)
