package reflection

import (
	"context"
	"fmt"
	pb "g-echo/reflection/proto"
	proto "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	testAddr = "127.0.0.1:50051"
	port     = 50051
	name     = "caowanyang"
	service  = "proto.Test"
)

type handler struct {
}

func (h *handler) GetUser(ctx context.Context, request *pb.UserRequest) (*pb.UserResponse, error) {
	return &pb.UserResponse{Name: request.GetName()}, nil
}

type server struct {
	s        *grpc.Server
	startErr chan error
}

func newServer() *server {
	return &server{startErr: make(chan error, 1)}
}

func (s *server) start(t *testing.T, p int) {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(p))
	if err != nil {
		s.startErr <- fmt.Errorf("failed to listen: %v", err)
		return
	}
	s.s = grpc.NewServer()
	pb.RegisterTestServer(s.s, &handler{})
	reflection.Register(s.s)
	s.startErr <- nil
	go s.s.Serve(lis)
}

func (s *server) stop() {
	s.s.Stop()
}

func (s *server) wait(t *testing.T, timeout time.Duration) {
	select {
	case err := <-s.startErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(timeout):
		t.Fatalf("Time out after %v waitting for server to be ready", timeout)
	}
}

func setUp(t *testing.T, p int) *server {
	s := newServer()
	s.start(t, p)
	s.wait(t, 2*time.Second)
	return s
}

func TestInvoke(t *testing.T) {
	s := setUp(t, port)
	req := &pb.UserRequest{Name: name}
	payLoad, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("proto marshal failed: %v", err)
	}
	r, err := Invoke(context.Background(), testAddr, "/proto.Test/GetUser", payLoad)
	if err != nil || strings.TrimSpace(string(r)) != name {
		t.Fatalf("reflection.Invoke(_, _, _, _) = %v, want %v", string(r), name)
	}
	s.stop()
}

func TestGetReflection(t *testing.T) {
	s := setUp(t, port)
	r, err := GetReflection(context.Background(), testAddr)
	print(strings.Join(r.Services, ","))
	if err != nil || strings.Join(r.Services, ",") != service {
		t.Fatalf("reflection.GetReflection(_, _) = %v, want %v", strings.Join(r.Services, ","), service)
	}
	s.stop()
}
