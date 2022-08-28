module sam/lambda

go 1.17

require (
	github.com/apex/gateway v1.1.2
	github.com/aws/aws-lambda-go v1.31.1
	github.com/aws/aws-sdk-go v1.44.86
	github.com/wei-zhong90/lambdautils v0.0.0
)

replace github.com/wei-zhong90/lambdautils => ./lambdautils

require (
	github.com/go-stomp/stomp v2.1.4+incompatible
	github.com/go-stomp/stomp/v3 v3.0.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)
