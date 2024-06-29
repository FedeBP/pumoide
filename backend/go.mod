module github.com/FedeBP/pumoide/backend

go 1.22.4

replace github.com/FedeBP/pumoide/backend => ./

require (
	github.com/aws/aws-sdk-go v1.54.10
	github.com/google/uuid v1.6.0
	github.com/kirsle/configdir v0.0.0-20170128060238-e45d2f54772f
	golang.org/x/time v0.5.0
)

require github.com/jmespath/go-jmespath v0.4.0 // indirect
