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
	"encoding/base64"
	"errors"
	"net/url"
    "log"
	"sync"
	"time"

	"github.com/acidvertigo/sessions/store"
	"github.com/valyala/fasthttp"
)

type (
	// IManager is the interface which Manager should implement
	IManager interface {
		Start(*fasthttp.RequestCtx) store.IStore
		Destroy(*fasthttp.RequestCtx)
		GC()
	}
	// Manager implements the IManager interface
	// contains the cookie's name, the provider and a duration for GC and cookie life expire
	Manager struct {
		cookieName string
		mu         sync.Mutex
		provider   IProvider
		gcDuration time.Duration
	}
)

var _ IManager = &Manager{}

var (
	continueOnError = true
	providers       = make(map[string]IProvider)
)

// newManager creates & returns a new Manager
// accepts 4 parameters
// first is the providerName (string) ["memory","redis"]
// second is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// third is the gcDuration (time.Duration) when this time passes it removes the sessions
// which hasn't be used for a long time(gcDuration), this number is the cookie life(expires) also
func newManager(providerName string, cookieName string, gcDuration time.Duration) (*Manager, error) {
	provider, found := providers[providerName]
	if !found {
		return nil, errors.New(providerName)
	}
	if gcDuration < 1 {
		gcDuration = time.Duration(60) * time.Minute
	}

	if cookieName == "" {
		cookieName = "AppCookieName"
	}

	manager := &Manager{}
	manager.provider = provider
	manager.cookieName = cookieName

	manager.gcDuration = gcDuration

	return manager, nil
}

// New creates & returns a new Manager and start its GC
// accepts 4 parameters
// first is the providerName (string) ["memory","redis"]
// second is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// third is the gcDuration (time.Duration) when this time passes it removes the sessions
// which hasn't be used for a long time(gcDuration), this number is the cookie life(expires) also
func New(providerName string, cookieName string, gcDuration time.Duration) *Manager {
	manager, err := newManager(providerName, cookieName, gcDuration)
	if err != nil {
		panic(err.Error()) // we have to panic here because we will start GC after and if provider is nil then many panics will come
	}
	//run the GC here
	go manager.GC()
	return manager
}

// Register registers a provider
func Register(provider IProvider) {
	if provider == nil {
		panic("no service provider")
	}
	providerName := provider.Name()

	if _, exists := providers[providerName]; exists {
		if !continueOnError {
			log.Fatal(providerName)
		} else {
			// do nothing it's a map it will overrides the existing provider.
		}
	}

	providers[providerName] = provider
}

// Manager implementation

func (m *Manager) generateSessionID() string {
	return base64.URLEncoding.EncodeToString(Random(32))
}

// Start starts the session
func (m *Manager) Start(requestCtx *fasthttp.RequestCtx) store.IStore {

	m.mu.Lock()
	var store store.IStore
	cookieValue := string(requestCtx.Request.Header.Cookie(m.cookieName))

	if cookieValue == "" { // cookie doesn't exists, let's generate a session and add set a cookie
		sid := m.generateSessionID()
		store, _ = m.provider.Init(sid)
		cookie := fasthttp.AcquireCookie()
		cookie.SetKey(m.cookieName)
		cookie.SetValue(url.QueryEscape(sid))
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		if ctx.IsTls {
		    cookie.SetDomain(os.Getenv
		}
		exp := time.Now().Add(m.gcDuration)
		cookie.SetExpire(exp)
		requestCtx.Response.Header.SetCookie(cookie)
		fasthttp.ReleaseCookie(cookie)
		//println("manager.go:156-> Setting cookie with lifetime: ", m.lifeDuration.Seconds())
	} else {
		sid, _ := url.QueryUnescape(cookieValue)
		store, _ = m.provider.Read(sid)
	}

	m.mu.Unlock()
	return store
}

// Destroy kills the session and remove the associated cookie
func (m *Manager) Destroy(requestCtx *fasthttp.RequestCtx) {
	cookieValue := string(requestCtx.Request.Header.Cookie(m.cookieName))
	if cookieValue == "" { // nothing to destroy
		return
	}

	m.mu.Lock()
	m.provider.Destroy(cookieValue)

	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(m.cookieName)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	if requestCtx.IsTLS() {
	    cookie.SetSecure(true)
	}
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but make sure 1 minute because we are using fasthttp
	cookie.SetExpire(exp)
	requestCtx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)

	m.mu.Unlock()
}

// GC tick-tock for the store cleanup
// it's a blocking function, so run it with go routine, it's totally safe
func (m *Manager) GC() {
	m.mu.Lock()

	m.provider.GC(m.gcDuration)
	// set a timer for the next GC
	time.AfterFunc(m.gcDuration, func() {
		m.GC()
	}) // or m.expire.Unix() if Nanosecond() doesn't works here
	m.mu.Unlock()
}
