module github.com/FedeBP/pumoide/backend

go 1.22.4

replace github.com/FedeBP/pumoide/backend => ./

require (
	github.com/aws/aws-sdk-go v1.54.15
	github.com/google/uuid v1.6.0
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.17.1
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/time v0.5.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/sys v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
