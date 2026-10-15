module github.com/sangay/web303/checkins

go 1.25.0

replace github.com/sangay/web303/proto => ../proto

require (
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
	github.com/sangay/web303/proto v0.0.0
	github.com/sony/gobreaker v1.0.0
	google.golang.org/grpc v1.84.0
	google.golang.org/protobuf v1.36.12
)

require (
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
)
