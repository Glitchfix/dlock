package lock

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type LockRequest struct {
	ID         string
	Acquirer   string
	Expiration time.Timer
	Next       *LockRequest
	Prev       *LockRequest
	Status     int
}

func (lq *LockRequest) Print() {
	head := lq
	s := "head"
	for head != nil {
		s += fmt.Sprintf("->%s", head.Acquirer)
		head = head.Next
	}

	log.Debug().Msgf("[LockRequest] %s", s)
}
