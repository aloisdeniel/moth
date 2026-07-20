module github.com/aloisdeniel/moth

go 1.25.12

require (
	connectrpc.com/connect v1.20.0
	connectrpc.com/cors v0.1.0
	connectrpc.com/grpchealth v1.5.0
	connectrpc.com/grpcreflect v1.3.0
	github.com/BurntSushi/toml v1.6.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/rs/cors v1.11.1
	github.com/spf13/cobra v1.10.2
	github.com/yuin/goldmark v1.7.17
	golang.org/x/crypto v0.54.0
	golang.org/x/term v0.45.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260715203245-bcc9394bd25e
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.53.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	modernc.org/libc v1.73.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
