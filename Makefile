
# These environment variables must be set for deployment to work.
S3_BUCKET := cf-templates-1nhvjnuy2vpok-ap-northeast-1
STACK_NAME := go-test

# Common values used throughout the Makefile, not intended to be configured.
TEMPLATE = template.yaml
PACKAGED_TEMPLATE = packaged.yaml

.PHONY: test
test: install
	go test -v ./lambdautils
	go test -v ./service/api

.PHONY: clean
clean:
	rm -f build/{api,error,worker} $(PACKAGED_TEMPLATE)

.PHONY: install
install:
	go get ./...

.PHONY: update
update:
	go get -u ./...

api: ./service/api/main.go
	go build -o build/api ./service/api
	
worker: ./service/worker/main.go
	go build -o build/worker ./service/worker


.PHONY: lambda
lambda:
	GOOS=linux GOARCH=amd64 $(MAKE) api
	GOOS=linux GOARCH=amd64 $(MAKE) worker

.PHONY: build
build: clean lambda

.PHONY: run
run: build
	sam local start-api

.PHONY: package
package: build
	sam package --template-file $(TEMPLATE) --s3-bucket $(S3_BUCKET) --output-template-file $(PACKAGED_TEMPLATE)

.PHONY: deploy
deploy: package
	sam deploy --stack-name $(STACK_NAME) --template-file $(PACKAGED_TEMPLATE) --capabilities CAPABILITY_IAM

.PHONY: teardown
teardown:
	aws cloudformation delete-stack --stack-name $(STACK_NAME)