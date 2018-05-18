package reflection

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type Message struct {
	Payload []byte
}

func (m *Message) Reset() {}

func (m *Message) String() string {
	return ""
}

func (m *Message) ProtoMessage() {}

func (m *Message) Marshal() ([]byte, error) {
	return m.Payload, nil
}

func (m *Message) Unmarshal(buf []byte) error {
	m.Payload = buf
	return nil
}

func Invoke(ctx context.Context, addr, method string, payload []byte) ([]byte, error) {
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	req := &Message{Payload: payload}
	res := &Message{}
	err = conn.Invoke(ctx, method, req, res)
	if err != nil {
		return nil, err
	}
	return res.Payload, nil

}

type Reflection struct {
	Services        []string
	FileDescriptors [][]byte
}

func GetReflection(ctx context.Context, addr string) (*Reflection, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	client := pb.NewServerReflectionClient(conn)

	stream, err := client.ServerReflectionInfo(ctx)

	if err != nil {
		return nil, err
	}

	services, err := getServices(stream)

	if err != nil {
		return nil, err
	}

	fileDescriptors, err := getFileDescriptors(stream, services)

	if err != nil {
		return nil, err
	}

	return &Reflection{
		Services:        services,
		FileDescriptors: fileDescriptors,
	}, nil
}

func getFileDescriptors(stream pb.ServerReflection_ServerReflectionInfoClient, services []string) ([][]byte, error) {
	for _, service := range services {
		err := stream.SendMsg(&pb.ServerReflectionRequest{
			MessageRequest: &pb.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: service,
			},
		})

		if err != nil {
			return nil, err
		}
	}

	r := make([][]byte, 0, len(services))

	for i := 0; i < len(services); i++ {
		res, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		if errorRes := res.GetErrorResponse(); errorRes != nil {
			return nil, errors.New(errorRes.ErrorMessage)
		}

		fileRes := res.GetFileDescriptorResponse()

		if fileRes == nil {
			return nil, errors.New("Unexpected reflection response")
		}

		r = append(r, fileRes.FileDescriptorProto...)
	}

	return r, nil
}

func getServices(stream pb.ServerReflection_ServerReflectionInfoClient) ([]string, error) {
	err := stream.SendMsg(&pb.ServerReflectionRequest{
		MessageRequest: &pb.ServerReflectionRequest_ListServices{},
	})

	if err != nil {
		return nil, err
	}

	res, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	if errorRes := res.GetErrorResponse(); errorRes != nil {
		return nil, errors.New(errorRes.ErrorMessage)
	}

	serviceRes := res.GetListServicesResponse()

	if serviceRes == nil {
		return nil, errors.New("Unexpected reflection response")
	}

	r := make([]string, 0, len(serviceRes.Service))

	for _, service := range serviceRes.Service {
		if service.Name != "grpc.reflection.v1alpha.ServerReflection" {
			r = append(r, service.Name)
		}
	}

	return r, nil
}
