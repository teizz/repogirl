package main

import (
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

func enable_debugging() {
	logrus.SetLevel(logrus.DebugLevel)
}

func makefields(ctx ...interface{}) (fields logrus.Fields) {
	fields = make(logrus.Fields)
	for i := 0; i < len(ctx); i += 2 {
		fields[ctx[i].(string)] = ctx[i+1]
	}
	return
}

func fatal(msg string, ctx ...interface{}) {
	logrus.WithFields(makefields(ctx...)).Fatal(msg)
	os.Exit(1)
}

func info(msg string, ctx ...interface{}) {
	logrus.WithFields(makefields(ctx...)).Info(msg)
}

func warn(msg string, ctx ...interface{}) {
	logrus.WithFields(makefields(ctx...)).Warn(msg)
}

func debug(msg string, ctx ...interface{}) {
	logrus.WithFields(makefields(ctx...)).Debug(msg)
}

func getcontextlogger(ctx ...interface{}) *log.Logger {
	return log.New(logrus.WithFields(makefields(ctx...)).Writer(), "", 0)
}
