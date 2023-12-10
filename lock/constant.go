package lock

const (
	LOCK_WAIT int64 = iota
	LOCK_NOT_ACQUIRED
	LOCK_ACQUIRED
	LOCK_DOUBLE_ACQUIRE
	LOCK_RELEASED
)

var (
	LockStatus = map[int64]string{
		LOCK_WAIT:           "LOCK_WAIT",
		LOCK_NOT_ACQUIRED:   "LOCK_NOT_ACQUIRED",
		LOCK_ACQUIRED:       "LOCK_ACQUIRED",
		LOCK_DOUBLE_ACQUIRE: "LOCK_DOUBLE_ACQUIRE",
		LOCK_RELEASED:       "LOCK_RELEASED",
	}
)
