package services

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/iamgak/grpc_gis/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type myGRPCServer struct {
	grpcSrv *grpc.Server
	// addr       string
	Gis *MadinaGisService
}

// == create new connection ==
func NewServer(service *MadinaGisService, logger *logrus.Logger) *myGRPCServer {

	return &myGRPCServer{
		grpcSrv: grpc.NewServer(grpc.UnaryInterceptor(LogRequestInterceptor(logger))),
		Gis:     service,
	}
}

func LogRequestInterceptor(logger *logrus.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {

		// Get client IP if available
		clientIP := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
			"client": clientIP,
		}).Info("Incoming gRPC request")

		// Call the actual handler
		resp, err = handler(ctx, req)

		// Get status code
		st, _ := status.FromError(err)

		logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
			"status": st.Code(),
		}).Info("gRPC request completed")

		return resp, err
	}
}

func (s *myGRPCServer) RegisterServices(register func(service *MadinaGisService, server *grpc.Server)) {
	register(s.Gis, s.grpcSrv)
}

// === Register Service Function ===
func registerGisService(service *MadinaGisService, server *grpc.Server) {
	pb.RegisterMadinaGisServiceServer(server, &MadinaGisGRPCServer{service: service})
	log.Println("All Service registered!")
}

func (s *myGRPCServer) Serve(ctx context.Context) {
	lis, err := net.Listen("tcp", s.Gis.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gRPC server...")
		s.grpcSrv.GracefulStop()
	}()

	log.Printf("gRPC server running on %s...", s.Gis.port)
	if err := s.grpcSrv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

// === Register services and serve it ===
func (g *myGRPCServer) Activate(ctx context.Context) {
	reflection.Register(g.grpcSrv)
	g.RegisterServices(registerGisService)
	g.Serve(ctx)
	fmt.Print("Hello")
}
