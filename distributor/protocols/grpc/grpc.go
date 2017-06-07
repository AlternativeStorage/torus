package grpc

import (
	"net"
	"net/url"
	//	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	//"github.com/alternative-storage/torus/jaeger"
	//"github.com/opentracing/opentracing-go"

	"github.com/alternative-storage/torus"
	"github.com/alternative-storage/torus/distributor/protocols"
	"github.com/alternative-storage/torus/models"

	//"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
)

const defaultPort = "40000"

func init() {
	protocols.RegisterRPCListener("http", grpcRPCListener)
	protocols.RegisterRPCDialer("http", grpcRPCDialer)
}

func grpcRPCListener(url *url.URL, hdl protocols.RPC, gmd torus.GlobalMetadata) (protocols.RPCServer, error) {
	out := &handler{
		handle: hdl,
	}
	h := url.Host
	if !strings.Contains(h, ":") {
		h = net.JoinHostPort(h, defaultPort)
	}
	lis, err := net.Listen("tcp", h)
	if err != nil {
		return nil, err
	}

	uIntOpt := grpc.UnaryInterceptor(grpc_opentracing.UnaryServerInterceptor())
	//sIntOpt := grpc_opentracing.StreamServerInterceptor()
	out.grpc = grpc.NewServer(uIntOpt)

	models.RegisterTorusStorageServer(out.grpc, out)
	go out.grpc.Serve(lis)
	return out, nil
}

func grpcRPCDialer(url *url.URL, timeout time.Duration, gmd torus.GlobalMetadata) (protocols.RPC, error) {
	h := url.Host
	if !strings.Contains(h, ":") {
		h = net.JoinHostPort(h, defaultPort)
	}
	uIntOpt := grpc.WithUnaryInterceptor(grpc_opentracing.UnaryClientInterceptor())
	conn, err := grpc.Dial(h, grpc.WithInsecure(), grpc.WithTimeout(timeout), uIntOpt)
	if err != nil {
		return nil, err
	}
	return &client{
		conn:    conn,
		handler: models.NewTorusStorageClient(conn),
	}, nil
}

type client struct {
	conn    *grpc.ClientConn
	handler models.TorusStorageClient
}

func (c *client) Close() error {
	return c.conn.Close()
}

// client (torusblk) calls this PutBlock.
func (c *client) PutBlock(ctx context.Context, ref torus.BlockRef, data []byte) error {
	_, err := c.handler.PutBlock(ctx, &models.PutBlockRequest{
		Refs: []*models.BlockRef{
			ref.ToProto(),
		},
		Blocks: [][]byte{
			data,
		},
	})
	return err
}

func (c *client) Block(ctx context.Context, ref torus.BlockRef) ([]byte, error) {
	resp, err := c.handler.Block(ctx, &models.BlockRequest{
		BlockRef: ref.ToProto(),
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *client) RebalanceCheck(ctx context.Context, refs []torus.BlockRef) ([]bool, error) {
	req := &models.RebalanceCheckRequest{}
	for _, x := range refs {
		req.BlockRefs = append(req.BlockRefs, x.ToProto())
	}
	resp, err := c.handler.RebalanceCheck(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Valid, nil
}

func (c *client) WriteBuf(ctx context.Context, ref torus.BlockRef) ([]byte, error) {
	panic("unimplemented")
}

type handler struct {
	handle protocols.RPC
	grpc   *grpc.Server
}

func (h *handler) Block(ctx context.Context, req *models.BlockRequest) (*models.BlockResponse, error) {
	data, err := h.handle.Block(ctx, torus.BlockFromProto(req.BlockRef))
	if err != nil {
		return nil, err
	}
	return &models.BlockResponse{
		Ok:   true,
		Data: data,
	}, nil
}

// This PutBlock is called by server(torusd) side.
func (h *handler) PutBlock(ctx context.Context, req *models.PutBlockRequest) (*models.PutResponse, error) {
	for i, ref := range req.Refs {
		err := h.handle.PutBlock(ctx, torus.BlockFromProto(ref), req.Blocks[i])
		if err != nil {
			return nil, err
		}
	}
	return &models.PutResponse{Ok: true}, nil
}

func (h *handler) RebalanceCheck(ctx context.Context, req *models.RebalanceCheckRequest) (*models.RebalanceCheckResponse, error) {
	check := make([]torus.BlockRef, len(req.BlockRefs))
	for i, x := range req.BlockRefs {
		check[i] = torus.BlockFromProto(x)
	}
	out, err := h.handle.RebalanceCheck(ctx, check)
	if err != nil {
		return nil, err
	}
	return &models.RebalanceCheckResponse{
		Valid: out,
	}, nil
}

func (h *handler) Close() error {
	h.grpc.Stop()
	return nil
}
