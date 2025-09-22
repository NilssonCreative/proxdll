// Package proxy provides a generic framework for creating proxy DLLs.
package proxy

import (
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

// Manager handles the loading of the original DLL and manages function pointers.
type Manager struct {
	originalDLL *windows.DLL
	procs       map[string]*windows.Proc
	mu          sync.RWMutex
}

// New creates a new proxy Manager for a given DLL.
// It loads the original DLL into memory.
func New(originalDllPath string) (*Manager, error) {
	dll, err := windows.LoadDLL(originalDllPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load original DLL at %s: %w", originalDllPath, err)
	}

	return &Manager{
		originalDLL: dll,
		procs:       make(map[string]*windows.Proc),
	}, nil
}

// GetOriginalFunc retrieves and caches a function from the original DLL.
func (m *Manager) GetOriginalFunc(funcName string) (*windows.Proc, error) {
	m.mu.RLock()
	proc, ok := m.procs[funcName]
	m.mu.RUnlock()

	if ok {
		return proc, nil
	}

	// If not cached, find it in the DLL
	foundProc, err := m.originalDLL.FindProc(funcName)
	if err != nil {
		return nil, fmt.Errorf("could not find function %s in original DLL: %w", funcName, err)
	}

	// Cache the proc
	m.mu.Lock()
	m.procs[funcName] = foundProc
	m.mu.Unlock()

	return foundProc, nil
}

// CallOriginal invokes the original function with the given arguments.
// It uses the modern `proc.Call()` method.
func (m *Manager) CallOriginal(funcName string, args ...uintptr) (r1, r2 uintptr, lastErr error) {
	proc, err := m.GetOriginalFunc(funcName)
	if err != nil {
		// This is a critical error as the function doesn't exist.
		// A panic is appropriate here because the proxy cannot fulfill its contract.
		panic(err)
	}

	return proc.Call(args...)
}

// Free unloads the original DLL. It should be called during cleanup.
func (m *Manager) Free() error {
	return m.originalDLL.Release()
}
