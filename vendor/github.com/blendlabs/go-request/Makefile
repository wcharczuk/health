all: test cover

test:
	@go test -v

cover:
	@go test -coverprofile=profile.cov
	@go tool cover -html=profile.cov
	@rm profile.cov
