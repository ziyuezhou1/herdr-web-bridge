//go:build !windows

package ipc

import (
	"context"
	"errors"
	"io"
	"time"
)

type Server struct{}

func Listen() (*Server, error) { return nil, errors.New("named pipe IPC is supported only on Windows") }
func (s *Server) Serve(context.Context, Handler) error { return errors.New("unsupported platform") }
func (s *Server) Close() error { return nil }
func dial(time.Duration) (io.ReadWriteCloser, error) { return nil, errors.New("unsupported platform") }
