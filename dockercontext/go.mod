module github.com/mdelapenya/docker-sdk-go/dockercontext

go 1.23.6

replace github.com/mdelapenya/docker-sdk-go/dockerconfig => ../dockerconfig

require (
	github.com/mdelapenya/docker-sdk-go/dockerconfig v0.1.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
