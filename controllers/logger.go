package controllers

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
)

var zl = zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()
var logger logr.Logger = zerologr.New(&zl)
