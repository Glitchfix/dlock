package client

import (
	"context"
	"io"

	"github.com/Glitchfix/dlock/lock"
	"github.com/Glitchfix/dlock/proto/plugin"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type Client struct {
	cli plugin.DLockClient
	m   map[string]map[string]int
}

func NewClient(cc grpc.ClientConnInterface) (client Client) {
	cli := plugin.NewDLockClient(cc)
	client = Client{
		cli: cli,
		m:   map[string]map[string]int{},
	}
	return
}

func (c *Client) Lock(id, acquirer string) {
	resp, err := c.cli.Lock(context.Background(), &plugin.LockRequest{
		Operation: plugin.OperationType_Lock,
		Id:        id,
		Acquirer:  acquirer,
	})
	if nil != err {
		log.Panic().Err(err)
		return
	}

	rs, err := resp.Recv()
	if err == io.EOF {
		return
	}
	if err != nil {
		log.Error().Err(err)
		return
	}

	if rs.Status != lock.LOCK_ACQUIRED {
		log.Panic().Err(err)
		return
	}

}

func (c *Client) Unlock(id, acquirer string) {
	resp, err := c.cli.Unlock(context.Background(), &plugin.LockRequest{
		Operation: plugin.OperationType_Unlock,
		Id:        id,
		Acquirer:  acquirer,
	})
	if nil != err {
		log.Panic().Err(err).Msgf("%s:%s failed to unlock", id, acquirer)
		return
	}

	rs, err := resp.Recv()
	log.Debug().Msgf("%s:%s response received status %s", id, acquirer, lock.LockStatus[rs.Status])

	if err != nil && err == io.EOF {
		log.Panic().Err(err).Msgf("%s:%s failed to receive lock response", id, acquirer)
		return
	}

	if rs.Status != lock.LOCK_RELEASED {
		log.Panic().Msgf("%s:%s failed to %s", id, acquirer, lock.LockStatus[rs.Status])
		return
	}

}
