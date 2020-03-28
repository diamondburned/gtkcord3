package gdbus

import "testing"

func TestGetSessionBus(t *testing.T) {
	c, err := GetSessionBusSync()
	if err != nil {
		t.Fatal("Failed to get session bus:", err)
	}

	if c == nil {
		t.Fatal("c == nil")
	}

	err = c.Notify(Notification{
		AppName: "gtkcord3",
		AppIcon: "user-available",
		Title:   "Notification title",
		Message: "Test",
	})

	if err != nil {
		t.Fatal("Failed to notify:", err)
	}
}
