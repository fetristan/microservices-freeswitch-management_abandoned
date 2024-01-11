	On dev computer :
	nano ~/bashrc and add :

	export GOPATH=$HOME/go
	export PATH=$PATH:$GOPATH/bin

	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	source ~/.bashrc
	protoc --go-grpc_out=. sessionsservice.proto
	protoc --go_out=. sessionsservice.proto
	

	
	git local config :
	[url "ssh://git@github.com/"]
	    insteadOf = https://github.com/
	
	go env -w GOPRIVATE=github.com/fetristan
	go mod tidy
	$env:GOOS = "linux"
	go build


	https://grpc.io/docs/protoc-installation/
	https://developers.google.com/protocol-buffers/docs/reference/go-generated