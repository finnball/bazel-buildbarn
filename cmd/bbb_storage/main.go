package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore/configuration"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
)

func main() {
	var (
		blobstoreConfig = flag.String("blobstore-config", "/config/blobstore.conf", "Configuration for blob storage")
	)
	flag.Parse()

	// Web server for metrics and profiling.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	// Storage access.
	blobAccess, _, err := configuration.CreateBlobAccessObjectsFromConfig(*blobstoreConfig, false)
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}
	rs := blobstore.NewRedisServer(blobAccess)

	// Redis server.
	sock, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal("Failed to create listening socket: ", err)
	}
	go func() {
		for {
			conn, err := sock.Accept()
			if err == nil {
				go rs.HandleConnection(context.Background(), conn)
			} else {
				log.Print("Failed to accept incoming connection: ", err)
			}
		}
	}()

	// RPC server.
	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	remoteexecution.RegisterContentAddressableStorageServer(s, cas.NewContentAddressableStorageServer(blobAccess))
	bytestream.RegisterByteStreamServer(s, blobstore.NewByteStreamServer(blobAccess, 1<<16))
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(s)

	sock, err = net.Listen("tcp", ":8982")
	if err != nil {
		log.Fatal("Failed to create listening socket: ", err)
	}
	if err := s.Serve(sock); err != nil {
		log.Fatal("Failed to serve RPC server: ", err)
	}
}
