//go:build linux
// +build linux

package machineid

const (
	dbusPath    = "/var/lib/dbus/machine-id"
	dbusPathEtc = "/etc/machine-id"
)

func machineID() (string, error) {
	id, err := os.ReadFile(dbusPath)
	if err != nil {
		// try fallback path
		id, err = os.ReadFile(dbusPathEtc)
		if err != nil {
			return "", err
		}
	}

	return trim(string(id)), nil
}
