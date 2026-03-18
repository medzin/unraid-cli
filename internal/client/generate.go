package client

//go:generate go run github.com/Khan/genqlient
//go:generate go run ../../cmd/generate-capabilities -schema ../../graphql/schema.graphql -mutations ../../graphql/mutations -out introspect_gen.go
