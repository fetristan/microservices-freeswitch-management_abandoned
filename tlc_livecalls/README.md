	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	protoc --go-grpc_out=. sessionsservice.proto
	protoc --go_out=. sessionsservice.proto
	
	git local config :
	[url "ssh://git@github.com/"]
	    insteadOf = https://github.com/
	
	go env -w GOPRIVATE=github.com/fetristan