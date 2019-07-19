package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
)

func TestNewID(t *testing.T) {
	for i := 0; i < 1000; i++ {
		id := model.NewID()
		if len(id) != 26 {
			t.Fatal("ids should be exactly 26 chars")
		}
	}
}
