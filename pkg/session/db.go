// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package session

import (
	"log"
	"sync"

	"github.com/gitbundle/modules/timeutil"
	"github.com/gitbundle/server/pkg/auth"

	"gitea.com/go-chi/session"
)

// DBStore represents a session store implementation based on the DB.
type DBStore struct {
	sid  string
	lock sync.RWMutex
	data map[interface{}]interface{}
}

// NewDBStore creates and returns a DB session store.
func NewDBStore(sid string, kv map[interface{}]interface{}) *DBStore {
	return &DBStore{
		sid:  sid,
		data: kv,
	}
}

// Set sets value to given key in session.
func (s *DBStore) Set(key, val interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data[key] = val
	return nil
}

// Get gets value by given key in session.
func (s *DBStore) Get(key interface{}) interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.data[key]
}

// Delete delete a key from session.
func (s *DBStore) Delete(key interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.data, key)
	return nil
}

// ID returns current session ID.
func (s *DBStore) ID() string {
	return s.sid
}

// Release releases resource and save data to provider.
func (s *DBStore) Release() error {
	// Skip encoding if the data is empty
	if len(s.data) == 0 {
		return nil
	}

	data, err := session.EncodeGob(s.data)
	if err != nil {
		return err
	}

	return auth.UpdateSession(s.sid, data)
}

// Flush deletes all session data.
func (s *DBStore) Flush() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data = make(map[interface{}]interface{})
	return nil
}

// DBProvider represents a DB session provider implementation.
type DBProvider struct {
	maxLifetime int64
}

// Init initializes DB session provider.
// connStr: username:password@protocol(address)/dbname?param=value
func (p *DBProvider) Init(maxLifetime int64, connStr string) error {
	p.maxLifetime = maxLifetime
	return nil
}

// Read returns raw session store by session ID.
func (p *DBProvider) Read(sid string) (session.RawStore, error) {
	s, err := auth.ReadSession(sid)
	if err != nil {
		return nil, err
	}

	var kv map[interface{}]interface{}
	if len(s.Data) == 0 || s.Expiry.Add(p.maxLifetime) <= timeutil.TimeStampNow() {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(s.Data)
		if err != nil {
			return nil, err
		}
	}

	return NewDBStore(sid, kv), nil
}

// Exist returns true if session with given ID exists.
func (p *DBProvider) Exist(sid string) bool {
	has, err := auth.ExistSession(sid)
	if err != nil {
		panic("session/DB: error checking existence: " + err.Error())
	}
	return has
}

// Destroy deletes a session by session ID.
func (p *DBProvider) Destroy(sid string) error {
	return auth.DestroySession(sid)
}

// Regenerate regenerates a session store from old session ID to new one.
func (p *DBProvider) Regenerate(oldsid, sid string) (_ session.RawStore, err error) {
	s, err := auth.RegenerateSession(oldsid, sid)
	if err != nil {
		return nil, err
	}

	var kv map[interface{}]interface{}
	if len(s.Data) == 0 || s.Expiry.Add(p.maxLifetime) <= timeutil.TimeStampNow() {
		kv = make(map[interface{}]interface{})
	} else {
		kv, err = session.DecodeGob(s.Data)
		if err != nil {
			return nil, err
		}
	}

	return NewDBStore(sid, kv), nil
}

// Count counts and returns number of sessions.
func (p *DBProvider) Count() int {
	total, err := auth.CountSessions()
	if err != nil {
		panic("session/DB: error counting records: " + err.Error())
	}
	return int(total)
}

// GC calls GC to clean expired sessions.
func (p *DBProvider) GC() {
	if err := auth.CleanupSessions(p.maxLifetime); err != nil {
		log.Printf("session/DB: error garbage collecting: %v", err)
	}
}

func init() {
	session.Register("db", &DBProvider{})
}
