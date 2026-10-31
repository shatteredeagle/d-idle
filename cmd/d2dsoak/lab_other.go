//go:build !unix

package main

import (
	"context"
	"errors"
)

type Lab interface {
	Start(ctx context.Context, seed int, logPath string) (sock string, err error)
	Stop() error
}

type procLab struct {
	Bin, Data, Home, Toggles string
	Class                    int
	Name                     string
	MaxLoad, MinFreeGB       float64
}

func (*procLab) Start(context.Context, int, string) (string, error) {
	return "", errors.New("d2dsoak runs on Linux (the worker) only")
}
func (*procLab) Stop() error { return nil }
