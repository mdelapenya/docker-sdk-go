module github.com/docker/docker-sdk-go/dockercontext

go 1.24.1

replace github.com/docker/docker-sdk-go/dockerconfig => ../dockerconfig

require (
	github.com/docker/docker-sdk-go/dockerconfig v0.1.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
