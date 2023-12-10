package lock

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

type LockQueue struct {
	debug     bool
	lock      *sync.Mutex
	acquirers map[string]*LockRequest
	Head      *LockRequest
	Last      *LockRequest
}

func (lq *LockQueue) IsHead(lreq *LockRequest) (status int64) {
	lq.lock.Lock()
	defer lq.lock.Unlock()

	if lq.Head.Acquirer != lreq.Acquirer {
		status = LOCK_WAIT
		return
	}
	return LOCK_ACQUIRED
}

func (lq *LockQueue) Expire(lreq *LockRequest) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s", r)
		}
	}()

	select {
	case <-lreq.Expiration.C:
		lq.Remove(lreq)
	}
}

func (lq *LockQueue) Remove(lreq *LockRequest) {
	lq.lock.Lock()
	defer lq.lock.Unlock()

	defer delete(lq.acquirers, lreq.Acquirer)

	lock, ok := lq.acquirers[lreq.Acquirer]
	if !ok {
		return
	}

	if lq.Head == lock {
		lq.Head = nil
	}

	if lq.Last == lock {
		lq.Last = nil
	}

	if lreq.Prev != nil {
		lreq.Prev.Next = lreq.Next
	}
}

func (lq *LockQueue) Print() {
	if !lq.debug {
		return
	}
	head := lq.Head
	s := "head"
	for head != nil {
		s += fmt.Sprintf("->%s", head.Acquirer)
		head = head.Next
	}

	log.Debug().Msgf("[QUEUE] %s", s)
}

func (lq *LockQueue) Queue(lreq *LockRequest) (status int64) {
	lq.lock.Lock()
	defer lq.lock.Unlock()

	_, ok := lq.acquirers[lreq.Acquirer]
	if ok {
		log.Debug().Caller().Msgf("acquirer %s is already holding a lock", lreq.Acquirer)
		status = LOCK_DOUBLE_ACQUIRE
		return
	}

	log.Debug().Caller().Msgf("[BEFORE] acquirer = %s lq.Head = %+v lq.Last = %+v", lreq.Acquirer, lq.Head, lq.Last)
	log.Debug().Caller().Msgf("[LockRequest BEFORE] lreq = %+v", lreq)

	if lq.Head == nil {
		lq.Head = lreq
		lq.Last = lreq
		log.Debug().Caller().Msgf("acquirer %s is already holding a lock", lreq.Acquirer)
		status = LOCK_ACQUIRED
		return
	}
	log.Debug().Caller().Msgf("[AFTER] acquirer = %s lq.Head = %+v lq.Last = %+v", lreq.Acquirer, lq.Head, lq.Last)
	log.Debug().Caller().Msgf("[LockRequest AFTER] lreq = %+v", lreq)

	if lq.Last != nil {
		lq.Last.Next = lreq
	}
	lq.acquirers[lreq.Acquirer] = lreq
	lq.Last = lreq
	status = LOCK_WAIT
	return
}

func (lq *LockQueue) Dequeue(lreq *LockRequest) (next *LockRequest, status int64) {

	lq.lock.Lock()
	defer lq.lock.Unlock()

	lq.Print()

	defer delete(lq.acquirers, lreq.Acquirer)

	log.Debug().Caller().Msgf("%s attempting dequeue", lreq.Acquirer)

	if lq.Head == nil {
		status = LOCK_NOT_ACQUIRED
		log.Debug().Caller().Msgf("%s head is empty", lreq.Acquirer)
		return
	}

	if lq.Head.Acquirer != lreq.Acquirer {
		status = LOCK_NOT_ACQUIRED
		log.Debug().Caller().Msgf("%s attempted to unlock but lock is acquired by %s", lq.Head.Acquirer, lreq.Acquirer)
		return
	}
	status = LOCK_RELEASED

	if lq.Last.Acquirer == lq.Head.Acquirer {
		lq.Head = nil
		lq.Last = nil
		return
	}

	lq.Head.Print()

	next = lq.Head.Next
	next.Prev = lq.Head.Prev
	lq.Head = next

	if lq.Head.Next == nil {
		lq.Last = lq.Head
	}

	lq.Head.Print()
	return
}
