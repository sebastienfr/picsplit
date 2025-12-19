package main

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInitLog(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		InitLog(true)

		if logrus.GetLevel() != logrus.DebugLevel {
			t.Errorf("expected DebugLevel, got %v", logrus.GetLevel())
		}
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		InitLog(false)

		if logrus.GetLevel() != logrus.WarnLevel {
			t.Errorf("expected WarnLevel, got %v", logrus.GetLevel())
		}
	})
}
