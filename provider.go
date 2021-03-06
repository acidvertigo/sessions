// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package sessions

import (
	"container/list"
	"sync"
	"time"

	"github.com/acidvertigo/sessions/store"
)

// IProvider the type which Provider must implement
type IProvider interface {
	Name() string
	Init(string) (store.IStore, error)
	Read(string) (store.IStore, error)
	Destroy(string) error
	Update(string) error
	GC(time.Duration)
}

type (
	// Provider implements the IProvider
	// contains the temp sessions memory, the store and some options for the cookies
	Provider struct {
		name               string
		mu                 sync.Mutex
		sessions           map[string]*list.Element // underline TEMPORARY memory store
		list               *list.List               // for GC
		NewStore           func(sessionId string, cookieLifeDuration time.Duration) store.IStore
		OnDestroy          func(store store.IStore) // this is called when .Destroy
		cookieLifeDuration time.Duration
	}
)

var _ IProvider = &Provider{}

// NewProvider returns a new empty Provider
func NewProvider(name string) *Provider {
	provider := &Provider{name: name, list: list.New()}
	provider.sessions = make(map[string]*list.Element, 0)
	return provider
}

// Init creates the store for the first time for this session and returns it
func (p *Provider) Init(sid string) (store.IStore, error) {
	p.mu.Lock()

	newSessionStore := p.NewStore(sid, p.cookieLifeDuration)

	elem := p.list.PushBack(newSessionStore)
	p.sessions[sid] = elem
	p.mu.Unlock()
	return newSessionStore, nil
}

// Read returns the store which sid parameter is belongs
func (p *Provider) Read(sid string) (store.IStore, error) {
	if elem, found := p.sessions[sid]; found {
		return elem.Value.(store.IStore), nil
	}
	// if not found
	sessionStore, err := p.Init(sid)
	return sessionStore, err

}

// Destroy always returns a nil error, for now.
func (p *Provider) Destroy(sid string) error {
	if elem, found := p.sessions[sid]; found {
		elem.Value.(store.IStore).Destroy()
		delete(p.sessions, sid)
		p.list.Remove(elem)
	}

	return nil
}

// Update updates the lastAccessedTime, and moves the memory place element to the front
// always returns a nil error, for now
func (p *Provider) Update(sid string) error {
	p.mu.Lock()

	if elem, found := p.sessions[sid]; found {
		elem.Value.(store.IStore).SetLastAccessedTime(time.Now())
		p.list.MoveToFront(elem)
	}

	p.mu.Unlock()
	return nil
}

// GC clears the memory
func (p *Provider) GC(duration time.Duration) {
	p.mu.Lock()
	p.cookieLifeDuration = duration
	defer p.mu.Unlock() //let's defer it and trust the go

	for {
		elem := p.list.Back()
		if elem == nil {
			break
		}

		// if the time has passed. session was expired, then delete the session and its memory place
		if (elem.Value.(store.IStore).LastAccessedTime().Unix() + duration.Nanoseconds()) < time.Now().Unix() {
			p.list.Remove(elem)
			delete(p.sessions, elem.Value.(store.IStore).ID())

		} else {
			break
		}
	}
}

// Name the provider's name, example: 'memory' or 'redis'
func (p *Provider) Name() string {
	return p.name
}
