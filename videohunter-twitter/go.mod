module github.com/victoraldir/myvideohuntertwitter

go 1.23.4

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go v1.55.5
	github.com/stretchr/testify v1.10.0
	github.com/victoraldir/myvideohuntershared v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/victoraldir/myvideohuntershared => ../videohunter-shared
