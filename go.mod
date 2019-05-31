module swarm-backup

require (
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.4
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092 // indirect
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v0.0.0-20180718150940-a3ef7e9a9bda
