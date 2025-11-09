module github.com/ncruces/go-sqlite3/litestream

go 1.24.4

require (
	github.com/benbjohnson/litestream v0.5.2
	github.com/ncruces/go-sqlite3 v0.30.1
	github.com/ncruces/wbt v0.2.0
	github.com/superfly/ltx v0.5.0
)

// github.com/ncruces/go-sqlite3
require (
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.10.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

// github.com/superfly/ltx
require github.com/pierrec/lz4/v4 v4.1.22 // indirect

// github.com/benbjohnson/litestream
require (
	filippo.io/age v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.2 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/psanford/sqlite3vfs v0.0.0-20240315230605-24e1d98cf361 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	modernc.org/sqlite v1.40.0 // indirect
)

// github.com/benbjohnson/litestream/s3
require (
	github.com/aws/aws-sdk-go-v2 v1.37.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.30.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.85.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.31.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.35.1 // indirect
	github.com/aws/smithy-go v1.22.5 // indirect
)

replace modernc.org/sqlite => github.com/ncruces/go-sqlite3/litestream/modernc v0.0.0-20251109124432-99b097de3b79
