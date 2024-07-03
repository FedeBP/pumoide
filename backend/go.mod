module github.com/FedeBP/pumoide/backend

go 1.22.4

replace github.com/FedeBP/pumoide/backend => ./

require (
	github.com/aws/aws-sdk-go v1.54.10
	github.com/google/uuid v1.6.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/time v0.5.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)
