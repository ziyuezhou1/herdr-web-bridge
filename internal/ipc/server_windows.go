//go:build windows

package ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	pipeAccessDuplex       = 0x00000003
	pipeTypeByte           = 0x00000000
	pipeReadModeByte       = 0x00000000
	pipeWait               = 0x00000000
	pipeRejectRemote       = 0x00000008
	pipeUnlimitedInstances = 255
	sddlRevision1          = 1
	errorPipeBusy          = syscall.Errno(231)
	errorPipeConnected     = syscall.Errno(535)
)

var (
	kernel32IPC  = syscall.NewLazyDLL("kernel32.dll")
	advapi32IPC  = syscall.NewLazyDLL("advapi32.dll")
	createPipeW  = kernel32IPC.NewProc("CreateNamedPipeW")
	connectPipe  = kernel32IPC.NewProc("ConnectNamedPipe")
	waitPipeW    = kernel32IPC.NewProc("WaitNamedPipeW")
	localFree    = kernel32IPC.NewProc("LocalFree")
	convertSDDLW = advapi32IPC.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
)

type Server struct {
	name    string
	sid     string
	mu      sync.Mutex
	current syscall.Handle
	closed  bool
}

func Listen() (*Server, error) {
	name, sid, err := PipeIdentity()
	if err != nil {
		return nil, err
	}
	return &Server{name: name, sid: sid}, nil
}

func (s *Server) Serve(ctx context.Context, handler Handler) error {
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	for {
		handle, err := createNamedPipe(s.name, s.sid)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = syscall.CloseHandle(handle)
			return nil
		}
		s.current = handle
		s.mu.Unlock()

		err = connectNamedPipe(handle)
		s.mu.Lock()
		if s.current == handle {
			s.current = 0
		}
		s.mu.Unlock()
		if err != nil {
			_ = syscall.CloseHandle(handle)
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		connection := os.NewFile(uintptr(handle), s.name)
		if connection == nil {
			_ = syscall.CloseHandle(handle)
			return errors.New("cannot wrap named pipe handle")
		}
		go serveConnection(connection, handler)
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.current != 0 {
		err := syscall.CloseHandle(s.current)
		s.current = 0
		return err
	}
	return nil
}

func createNamedPipe(name, sid string) (syscall.Handle, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	sddlPtr, err := syscall.UTF16PtrFromString("D:P(A;;GA;;;" + sid + ")")
	if err != nil {
		return 0, err
	}
	var descriptor uintptr
	result, _, callErr := convertSDDLW.Call(
		uintptr(unsafe.Pointer(sddlPtr)),
		uintptr(sddlRevision1),
		uintptr(unsafe.Pointer(&descriptor)),
		0,
	)
	if result == 0 {
		return 0, fmt.Errorf("convert pipe security descriptor: %w", callErr)
	}
	defer localFree.Call(descriptor)
	attributes := syscall.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(syscall.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
		InheritHandle:      0,
	}
	handleValue, _, callErr := createPipeW.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(pipeAccessDuplex),
		uintptr(pipeTypeByte|pipeReadModeByte|pipeWait|pipeRejectRemote),
		uintptr(pipeUnlimitedInstances),
		uintptr(protocolBufferSize()),
		uintptr(protocolBufferSize()),
		0,
		uintptr(unsafe.Pointer(&attributes)),
	)
	handle := syscall.Handle(handleValue)
	if handle == syscall.InvalidHandle {
		return 0, fmt.Errorf("create named pipe: %w", callErr)
	}
	return handle, nil
}

func connectNamedPipe(handle syscall.Handle) error {
	result, _, callErr := connectPipe.Call(uintptr(handle), 0)
	if result != 0 || errors.Is(callErr, errorPipeConnected) {
		return nil
	}
	return fmt.Errorf("connect named pipe: %w", callErr)
}

func protocolBufferSize() uint32 { return 64 << 10 }

func dial(timeout time.Duration) (io.ReadWriteCloser, error) {
	name, _, err := PipeIdentity()
	if err != nil {
		return nil, err
	}
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		handle, openErr := syscall.CreateFile(
			namePtr,
			syscall.GENERIC_READ|syscall.GENERIC_WRITE,
			0,
			nil,
			syscall.OPEN_EXISTING,
			0,
			0,
		)
		if openErr == nil {
			file := os.NewFile(uintptr(handle), name)
			if file == nil {
				_ = syscall.CloseHandle(handle)
				return nil, errors.New("cannot wrap named pipe client handle")
			}
			return file, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("dial named pipe: %w", openErr)
		}
		remaining := time.Until(deadline)
		waitMillis := uint32(min(remaining.Milliseconds(), int64(100)))
		if waitMillis == 0 {
			waitMillis = 1
		}
		if errors.Is(openErr, errorPipeBusy) {
			waitPipeW.Call(uintptr(unsafe.Pointer(namePtr)), uintptr(waitMillis))
		} else {
			time.Sleep(time.Duration(waitMillis) * time.Millisecond)
		}
	}
}
