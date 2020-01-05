package cmd

import (
	"auth-middleware/pkg/protocol/grpc"
	v1 "auth-middleware/pkg/service/v1"
	"context"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v7"
)

// Config is configuration for Server
type Config struct {
	// gRPC server start parameters section
	// gRPC is TCP port to listen by gRPC server
	GRPCPort string
	// DB Datastore parameters section
	// DatastoreDBHost is host of database
	DatastoreDBHost string
	// DatastoreDBPassword password to connect to database
	DatastoreDBPassword string

	MaxRetries int
}

// RunServer runs gRPC server and HTTP gateway
func RunServer() error {
	ctx := context.Background()

	// get configuration
	var cfg Config
	flag.StringVar(&cfg.GRPCPort, "grpc-port", "", "gRPC port to bind")
	flag.StringVar(&cfg.DatastoreDBHost, "db-host", "", "Database host")
	flag.StringVar(&cfg.DatastoreDBPassword, "db-password", "", "Database password")
	flag.IntVar(&cfg.MaxRetries,"db-maxretries", 0, "Database Max Retries")
	flag.Parse()

	if len(cfg.GRPCPort) == 0 {
		return fmt.Errorf("invalid TCP port for gRPC server: '%s'", cfg.GRPCPort)
	}

	options := redis.Options{
		Addr:               cfg.DatastoreDBHost,
		Password:           cfg.DatastoreDBPassword,
		MaxRetries:         cfg.MaxRetries,
	}
	makePing(&options)
	v1API := v1.NewAuthServiceServer(&options)
	return grpc.RunServer(ctx, v1API, cfg.GRPCPort)
}

func makePing (options *redis.Options) {
	client := redis.NewClient(options)
	defer client.Close()
	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
}