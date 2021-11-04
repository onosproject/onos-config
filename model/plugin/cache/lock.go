// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugincache

import (
	"context"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"os"
	"sync"
	"syscall"
	"time"
)

// newPluginLock creates a new plugin file lock
func newPluginLock(path string) *pluginLock {
	return &pluginLock{
		path: path,
	}
}

// pluginLock is a plugin file lock
type pluginLock struct {
	path    string
	rlocked bool
	wlocked bool
	fh      *os.File
	mu      sync.RWMutex
}

// Lock acquires a write lock on the cache
func (l *pluginLock) Lock(ctx context.Context) error {
	locked, err := l.lock(ctx, &l.wlocked, syscall.LOCK_EX)
	if err != nil {
		err = errors.NewInternal(err.Error())
		log.Error(err)
		return err
	} else if !locked {
		err = errors.NewConflict("failed to acquire cache lock")
		log.Error(err)
		return err
	}
	return nil
}

// IsLocked checks whether the cache is write locked
func (l *pluginLock) IsLocked() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.wlocked
}

// Unlock releases a write lock from the cache
func (l *pluginLock) Unlock(ctx context.Context) error {
	if err := l.unlock(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// RLock acquires a read lock on the cache
func (l *pluginLock) RLock(ctx context.Context) error {
	locked, err := l.lock(ctx, &l.rlocked, syscall.LOCK_SH)
	if err != nil {
		err = errors.NewInternal(err.Error())
		log.Error(err)
		return err
	} else if !locked {
		err = errors.NewConflict("failed to acquire cache lock")
		log.Error(err)
		return err
	}
	return nil
}

// IsRLocked checks whether the cache is read locked
func (l *pluginLock) IsRLocked() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.wlocked || l.rlocked
}

// RUnlock releases a read lock on the cache
func (l *pluginLock) RUnlock(ctx context.Context) error {
	if err := l.unlock(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// lock attempts to acquire a file lock
func (l *pluginLock) lock(ctx context.Context, locked *bool, flag int) (bool, error) {
	for {
		if ok, err := l.tryLock(locked, flag); ok || err != nil {
			return ok, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(lockAttemptDelay):
			// try again
		}
	}
}

func (l *pluginLock) tryLock(locked *bool, flag int) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if *locked {
		return true, nil
	}

	if l.fh == nil {
		if err := l.openFH(); err != nil {
			return false, err
		}
		defer l.ensureFhState()
	}

	err := syscall.Flock(int(l.fh.Fd()), flag|syscall.LOCK_NB)
	switch err {
	case syscall.EWOULDBLOCK:
		return false, nil
	case nil:
		*locked = true
		return true, nil
	}
	return false, err
}

func (l *pluginLock) unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if (!l.wlocked && !l.rlocked) || l.fh == nil {
		return nil
	}

	if err := syscall.Flock(int(l.fh.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}

	l.fh.Close()

	l.wlocked = false
	l.rlocked = false
	l.fh = nil
	return nil
}

func (l *pluginLock) openFH() error {
	fh, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDONLY, os.FileMode(0666))
	if err != nil {
		return err
	}
	l.fh = fh
	return nil
}

func (l *pluginLock) ensureFhState() {
	if !l.wlocked && !l.rlocked && l.fh != nil {
		l.fh.Close()
		l.fh = nil
	}
}
