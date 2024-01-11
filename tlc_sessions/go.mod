module github.com/fetristan/tlc_sessions

go 1.18

require github.com/fetristan/tlc_logger v1.0.0

require (
	github.com/cgrates/fsock v0.0.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.6.0
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)

require (
	github.com/golang/protobuf v1.5.3
	github.com/fetristan/tlc_events v1.1.5
	golang.org/x/net v0.0.0-20220403103023-749bd193bc2b // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220401170504-314d38edb7de // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/cgrates/fsock => github.com/fetristan/tlc_fsock v1.2.0
