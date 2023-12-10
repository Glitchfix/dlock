package lock

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type LockManager struct {
	lock *sync.Mutex
	lmap map[string]*LockQueue
}

func NewLockManager() *LockManager {
	l := &LockManager{
		lock: &sync.Mutex{},
		lmap: map[string]*LockQueue{},
	}

	return l
}

func (l *LockManager) GetQueue(lreq *LockRequest) (queue *LockQueue) {
	l.lock.Lock()
	defer l.lock.Unlock()
	queue, ok := l.lmap[lreq.ID]
	if !ok {
		queue = &LockQueue{
			lock:      &sync.Mutex{},
			acquirers: map[string]*LockRequest{},
			Head:      nil,
			Last:      nil,
		}
		l.lmap[lreq.ID] = queue
	}
	return
}

func (l *LockManager) Lock(lreq *LockRequest) (status int64) {
	queue := l.GetQueue(lreq)

	status = queue.Queue(lreq)

	log.Debug().Caller().Msgf("acquirer %s has %s on %s", lreq.Acquirer, LockStatus[status], lreq.ID)
	return
}

func (l *LockManager) Remove(lreq *LockRequest) {
	var queue *LockQueue
	l.lock.Lock()
	defer l.lock.Unlock()
	queue, ok := l.lmap[lreq.ID]
	if !ok {
		return
	}

	queue.Remove(lreq)
}

func (l *LockManager) IsHead(lreq *LockRequest) (status int64) {
	var queue *LockQueue
	l.lock.Lock()
	defer l.lock.Unlock()
	queue, ok := l.lmap[lreq.ID]
	if !ok {
		return
	}

	return queue.IsHead(lreq)
}

func (l *LockManager) Unlock(lreq *LockRequest) (next *LockRequest, status int64) {
	var queue *LockQueue
	l.lock.Lock()
	defer l.lock.Unlock()
	log.Debug().Msgf("[Unlock] l.lmap: %+v lreq.ID: %s", l.lmap, lreq.ID)
	queue, ok := l.lmap[lreq.ID]

	if ok {
		next, status = queue.Dequeue(lreq)

		log.Debug().Caller().Msgf("unlocked %s has %s on %s", lreq.Acquirer, LockStatus[status], lreq.ID)
		return
	}
	status = LOCK_NOT_ACQUIRED
	return
}
