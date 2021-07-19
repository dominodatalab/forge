package bkimage

import (
	"context"
	"time"

	controlapi "github.com/moby/buildkit/api/services/control"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func (c *Client) Solve(ctx context.Context, req *controlapi.SolveRequest, ch chan *controlapi.StatusResponse) error {
	defer close(ch)

	if c.controller == nil {
		if err := c.createController(ctx); err != nil {
			return err
		}
	}

	statusCtx, cancelStatus := context.WithCancel(ctx)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer func() {
			go func() {
				<-time.After(3 * time.Second)
				cancelStatus()
			}()
		}()

		_, err := c.controller.Solve(ctx, req)
		if err != nil {
			err = errors.Wrap(err, "failed to solve")
		}
		return err
	})
	eg.Go(func() error {
		statusReq := &controlapi.StatusRequest{
			Ref: req.Ref,
		}
		server := &controlStatusServer{
			ctx: statusCtx,
			ch:  ch,
		}

		return c.controller.Status(statusReq, server)
	})
	return eg.Wait()
}

type controlStatusServer struct {
	ctx context.Context
	ch  chan *controlapi.StatusResponse

	grpc.ServerStream
}

func (s controlStatusServer) Send(resp *controlapi.StatusResponse) error {
	s.ch <- resp
	return nil
}

func (s *controlStatusServer) SendMsg(m interface{}) error {
	return s.Send(m.(*controlapi.StatusResponse))
}

func (s *controlStatusServer) Context() context.Context {
	return s.ctx
}
