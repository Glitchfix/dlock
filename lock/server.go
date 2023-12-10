package lock

import (
	"sync"

	"github.com/Glitchfix/dlock/proto/plugin"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog/log"
)

type Server struct {
	ID   string
	lock *sync.Mutex
	*plugin.UnimplementedDLockServer
	manager  *LockManager
	notifier map[string]map[string]chan *plugin.LockResponse

	raftInstance *raft.Raft
	blankSink    *LockServerBlankSink
}

func NewServer() plugin.DLockServer {
	s := &Server{
		lock:     &sync.Mutex{},
		manager:  NewLockManager(),
		notifier: map[string]map[string]chan *plugin.LockResponse{},
	}

	s.Boot()

	return s
}

func (s *Server) getNotifier(lreq *LockRequest) (c chan *plugin.LockResponse) {
	if lreq == nil {
		return nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.notifier[lreq.ID]
	if !ok {
		s.notifier[lreq.ID] = map[string]chan *plugin.LockResponse{}
	}
	_, ok = s.notifier[lreq.ID][lreq.Acquirer]
	if !ok {
		s.notifier[lreq.ID][lreq.Acquirer] = make(chan *plugin.LockResponse, 1)
	}
	c = s.notifier[lreq.ID][lreq.Acquirer]
	return
}

func (s *Server) Lock(lreq *plugin.LockRequest, lres plugin.DLock_LockServer) (err error) {
	notifier := s.getNotifier(&LockRequest{
		ID:       lreq.Id,
		Acquirer: lreq.Acquirer,
	})
	abandon := true

	lockRQ := &LockRequest{
		ID:       lreq.Id,
		Acquirer: lreq.Acquirer,
	}

	status := s.manager.Lock(lockRQ)

	defer log.Debug().Msgf("lock acquired %s has %s on %s\n", lreq.Acquirer, LockStatus[status], lreq.Id)
	if status == LOCK_ACQUIRED {
		err = lres.Send(&plugin.LockResponse{
			Status: status,
		})
		if nil != err {
			log.Error().Err(err).Msg("failed to send lock response")
			return
		}
		return
	}

	log.Debug().Msgf("[SERVER] acquirer %s has %s on %s\n", lreq.Acquirer, LockStatus[status], lreq.Id)

	if !(status == LOCK_ACQUIRED || status == LOCK_WAIT) {
		log.Debug().Msgf("waiting for lock %s has %s on %s\n", lreq.Acquirer, LockStatus[status], lreq.Id)

		err = lres.Send(&plugin.LockResponse{
			Status: status,
		})
		return
	}

	defer func() {
		if abandon {
			s.manager.Remove(lockRQ)
		}
	}()

	err = lres.Send(<-notifier)
	if nil != err {
		log.Error().Err(err).Msg("failed to send lock response")
		return
	}
	abandon = false

	return
}

func (s *Server) Unlock(lreq *plugin.LockRequest, lres plugin.DLock_UnlockServer) (err error) {
	var status int64

	defer func() {
		s.manager.Remove(&LockRequest{
			ID:       lreq.Id,
			Acquirer: lreq.Acquirer,
		})
	}()
	log.Debug().Msgf("%s will try to unlock on %s", lreq.Acquirer, lreq.Id)

	var nextLreq *LockRequest

	nextLreq, status = s.manager.Unlock(&LockRequest{
		ID:       lreq.Id,
		Acquirer: lreq.Acquirer,
	})
	if status == LOCK_RELEASED {
		if nextLreq != nil {
			notifier := s.getNotifier(nextLreq)
			if notifier == nil {
				return
			}

			log.Debug().Msgf("[NOTIFIER] %s is notifying %s", lreq.Acquirer, nextLreq.Acquirer)
			notifier <- &plugin.LockResponse{
				Id:       nextLreq.ID,
				Acquirer: nextLreq.Acquirer,
				Status:   status,
			}

		}
	}

	err = lres.Send(&plugin.LockResponse{
		Id:       lreq.Id,
		Acquirer: lreq.Acquirer,
		Status:   status,
	})
	if nil != err {
		log.Error().Err(err).Msg("failed to send lock response")
		return
	}

	return
}
