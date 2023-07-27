package machineid

import "sync"

var (
	id    string
	idErr error
	once  sync.Once
)

func ID() (string, error) {
	return machineID()
}

func IDOnce() (string, error) {
	once.Do(func() {
		id, idErr = machineID()
	})

	return id, idErr
}
