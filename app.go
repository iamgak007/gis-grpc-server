package main

import (
	"context"

	"github.com/iamgak/grpc_gis/services"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type GisApp struct {
	Log        *zap.Logger
	GrpcServer grpcServer
}

type grpcServer interface {
	RegisterServices(func(gis *services.MadinaGisService, server *grpc.Server))
	Activate(ctx context.Context)
}

func Init(config *services.MadinaGisService, logger *logrus.Logger) *GisApp {
	app := &GisApp{
		GrpcServer: services.NewServer(config, logger),
	}
	return app
}
