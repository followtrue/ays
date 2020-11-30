package server

import (
	"ays/src/libs/constant"
	"ays/src/modules/listen"
	"ays/src/modules/logger"
	pb "ays/src/modules/rpc/proto"
	"ays/src/modules/tools"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
)

type Server struct{}

func GetGrpcServer() *grpc.Server {
	return grpc.NewServer()
}

func Start(rpcServer *grpc.Server, listener net.Listener){
	//创建服务器的一个实例
	rpcServer = grpc.NewServer()

	//注册job服务端
	pb.RegisterJobServer(rpcServer, &Server{})

	//阻塞等待，直到进程被杀死或者 Stop() 被调用
	err := rpcServer.Serve(listener)
	logger.Fatal("rpc server exit:", err)
}

func (s Server) Run(ctx context.Context, req *pb.JobRequest) (*pb.JobResponse, error) {
	defer func() {
		if err := recover(); err != nil {
			grpclog.Info(err)
		}
	}()

	resp := new(pb.JobResponse)
	//type
	if req.Type == constant.JobType {
		output, err := tools.ExecShell(ctx, req.Command)
		if len(output) > 1024 * 1024 {
			output = string([]rune(output)[:1024])
		}
		resp.Output = output
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Error = ""
		}


	} else if req.Type == constant.QueueType {
		// 被通知监听队列
		resp.Error = ""
		go func() {
			listen.ListenQueue(req.Command)
		}()
	}
	return resp, nil
}