package main

import (
	log "github.com/sirupsen/logrus"
)

var logger log.FieldLogger

func init() {
	logger = log.New()
}
