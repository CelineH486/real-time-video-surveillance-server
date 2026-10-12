package controllers

import (
	"testing"
	"time"

	"real-time-video-surveillance-system/models"
)

func TestLocationBroadcastDeliversNewestUpdate(t *testing.T) {
	controller := &LocationController{subscribers: make(map[string]map[chan models.TruckLocation]struct{})}
	updates, unsubscribe := controller.subscribe("truck001")
	defer unsubscribe()

	controller.broadcast(models.TruckLocation{TruckID: "truck001", Latitude: 24.1})
	controller.broadcast(models.TruckLocation{TruckID: "truck001", Latitude: 24.2})

	select {
	case location := <-updates:
		if location.Latitude != 24.2 {
			t.Fatalf("got latitude %f, want newest latitude 24.2", location.Latitude)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for location update")
	}
}

func TestLocationController_SubscribeAndUnsubscribe(t *testing.T) {
	controller := NewLocationController(nil)

	truckID := "test-truck-999"

	updates, unsubscribe := controller.subscribe(truckID)

	if updates == nil {
		t.Error("預期拿到一個頻道，卻得到 nil")
	}

	subCount := len(controller.subscribers[truckID])

	if subCount != 1 {
		t.Errorf("預期訂閱者數量為 1，卻得到 %d", subCount)
	}

	unsubscribe()

	_, exists := controller.subscribers[truckID]

	if exists {
		t.Error("預期取消訂閱後會清除 truckID 的紀錄，但它仍然存在 (可能造成記憶體洩漏)")
	}
}
