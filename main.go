package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	isClient              bool
	addr                  string
	interval              time.Duration
	maxConnectionIdle     time.Duration
	maxConnectionAge      time.Duration
	maxConnectionAgeGrace time.Duration
)

func main() {
	flag.BoolVar(&isClient, "client", false, "Whether to start the gRPC server or the gRPC client")
	flag.StringVar(&addr, "addr", "0.0.0.0:8888", "tcp host:port to connect or serve")
	flag.DurationVar(&interval, "client.interval", 3*time.Second, "The interval time to request to the server")
	flag.DurationVar(&maxConnectionIdle, "server.max-connection-idle", 0, "A duration for the amount of time after which an idle connection would be closed by sending a GoAway. Idleness duration is defined since the most recent time the number of outstanding RPCs became zero or the connection establishment (default infinity)")
	flag.DurationVar(&maxConnectionAge, "server.max-connection-age", 0, "A duration for the maximum amount of time a onnection may exist before it will be closed by sending a GoAway (default infinity)")
	flag.DurationVar(&maxConnectionAgeGrace, "server.max-connection-age-grace", 0, "An additive period after MaxConnectionAge after which the connection will be forcibly closed. (default infinity)")
	flag.Parse()

	log.Printf("Starting grpc hello: %+v", os.Args)

	if isClient {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
		conn, err := grpc.DialContext(ctx, addr, opts...)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		client := healthpb.NewHealthClient(conn)

		t := time.NewTicker(interval)
		defer t.Stop()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

	L:
		for {
			select {
			case <-t.C:
				go func() {
					var header metadata.MD
					resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{}, grpc.Header(&header))
					if err != nil {
						log.Printf("error: health rpc failed %+v", err)
					}
					log.Printf("resp: %v header: %+v", resp.String(), header)
				}()
			case <-c:
				break L
			}
		}
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}

		lis, err := net.Listen("tcp4", addr)
		if err != nil {
			log.Fatal(err)
		}

		var kpServerParams keepalive.ServerParameters
		if maxConnectionIdle != 0 {
			kpServerParams.MaxConnectionAge = maxConnectionIdle
		}
		if maxConnectionAge != 0 {
			kpServerParams.MaxConnectionAge = maxConnectionAge
		}
		if maxConnectionAgeGrace != 0 {
			kpServerParams.MaxConnectionAgeGrace = maxConnectionAgeGrace
		}

		server := grpc.NewServer(grpc.KeepaliveParams(kpServerParams))
		healthpb.RegisterHealthServer(server, &healthServer{hostname})
		reflection.Register(server)

		log.Printf("Start listening the grpc server on %s\n", lis.Addr().String())
		log.Fatal(server.Serve(lis))
	}
}

type healthServer struct {
	hostname string
}

func (h *healthServer) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	header := metadata.Pairs("hostname", h.hostname)
	grpc.SetHeader(ctx, header)

	return &healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	}, nil
}

func (h *healthServer) Watch(*healthpb.HealthCheckRequest, healthpb.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not implemented.")
}
