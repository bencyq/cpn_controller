package version2

import "testing"

func TestClient(t *testing.T) {
	var monitor Monitor
	monitor.clientForScheduler()
}
