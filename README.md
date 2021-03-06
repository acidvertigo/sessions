This is just a fork of [fasthttp-contrib/sessions](https://github.com/fasthttp-contrib/sessions/) with some modification to make it work on my personal framework

# Package information

This package is new and unique, if you notice a bug or issue [post it here](https://github.com/fasthttp-contrib/sessions/issues)


- Cleans the temp memory when a sessions is iddle, and re-loccate it , fast, to the temp memory when it's necessary. Also most used/regular sessions are going front in the memory's list.

- Supports redisstore and normal memory routing. If redisstore is used but fails to connect then ,automatically, switching to the memory storage.


**A session can be defined as a server-side storage of information that is desired to persist throughout the user's interaction with the web site** or web application.

Instead of storing large and constantly changing information via cookies in the user's browser, **only a unique identifier is stored on the client side** (called a "session id"). This session id is passed to the web server every time the browser makes an HTTP request (ie a page link or AJAX request). The web application pairs this session id with it's internal database/memory and retrieves the stored variables for use by the requested page.

----


## How to use

```go
// New creates & returns a new Manager and start its GC
// accepts 4 parameters
// first is the providerName (string) "memory" or "redis"
// second is the cookieName, the session's name (string) ["mysessionsecretcookieid"]
// third is the gcDuration (time.Duration)
// when this time passes it removes from
// temporary memory GC the value which hasn't be used for a long time(gcDuration)
// this is for the client's/browser's Cookie life time(expires) also

New(provider string, cName string, gcDuration time.Duration) *sessions.Manager

```

Example **memory**

```go

package main

import (
	"time"

	"github.com/valyala/fasthttp"
	"github.com/fasthttp-contrib/sessions"

	_ "github.com/fasthttp-contrib/sessions/providers/memory" // here we add the memory provider and its store
)

var sess *sessions.Manager

func init() {
	sess = sessions.New("memory", "yoursessionid_cookiename", time.Duration(60)*time.Minute)
}

func main() {


	requestHandler := func(ctx *fasthttp.RequestCtx) {
  		switch string(ctx.Path()) {
  		case "/set":
      		//get the session for this context
			session := sess.Start(ctx)

			//set session values
			session.Set("name", "anyName")

			//test if setted here
			ctx.SetBodyString("All ok session setted to "+session.GetString("name"))
  		case "/get":
      		//get the session for this context
			session := sess.Start(ctx)

			var name string

			//get the session value
			if v := session.Get("name"); v != nil {
				name = v.(string)
			}
			// OR just name = session.GetString("name")

			ctx.SetBodyString("The name on the /set was: "+ name)
  		case "/delete":
      		//get the session for this context
			session := sess.Start(ctx)

			session.Delete("name")
  		case "/clear":
      		//get the session for this context
			session := sess.Start(ctx)
			// removes all entries
			session.Clear()
  		case "/destroy":
      		//destroy, removes the entire session and cookie
			sess.Destroy(ctx)
  		}
	}


	fasthttp.ListenAndServe(":8080", requestHandler)
}

// session.GetAll() returns all values a map[interface{}]interface{}
// session.VisitAll(func(key interface{}, value interface{}) { /* loops for each entry */})

}



```


Example **redis** with default configuration

The default redis client points to 127.0.0.1:6379

```go

package main

import (
	"time"

	"github.com/fasthttp-contrib/sessions"

	_ "github.com/fasthttp-contrib/sessions/providers/redis"
    // here we add the redis  provider and its store
    // with the default redis client points to 127.0.0.1:6379
)

var sess *sessions.Manager

func init() {
	sess = sessions.New("redis", "yoursessionid", time.Duration(60)*time.Minute)
}

//... usage: same as memory example
```

Example **redis** with custom configuration
```go
type Config struct {
	// Network "tcp"
	Network string
	// Addr "127.0.01:6379"
	Addr string
	// Password string .If no password then no 'AUTH'. Default ""
	Password string
	// If Database is empty "" then no 'SELECT'. Default ""
	Database string
	// MaxIdle 0 no limit
	MaxIdle int
	// MaxActive 0 no limit
	MaxActive int
	// IdleTimeout 5 * time.Minute
	IdleTimeout time.Duration
	//Prefix "myprefix-for-this-website". Default ""
	Prefix string
	// MaxAgeSeconds how much long the redis should keep the session in seconds. Default 2520.0 (42minutes)
	MaxAgeSeconds int
}
```

```go
package main

import (
	"time"
	"github.com/fasthttp-contrib/sessions"

     "github.com/fasthttp-contrib/sessions/providers/redis"
)

var sess *sessions.Manager

func init() {
    // you can config the redis after init also, but before any client's request
    // but it's always a good idea to do it before sessions.New...
    redis.Config.Network = "tcp"
    redis.Config.Addr = "127.0.0.1:6379"
    redis.Config.Prefix = "myprefix-for-this-website"

	sess = sessions.New("redis", "irissessionid", time.Duration(60)*time.Minute)
}

//...usage: same as memory example
```

### Security: Prevent session hijacking

> This section  is external


**cookie only and token**

Through this simple example of hijacking a session, you can see that it's very dangerous because it allows attackers to do whatever they want. So how can we prevent session hijacking?

The first step is to only set session ids in cookies, instead of in URL rewrites. Also, we should set the httponly cookie property to true. This restricts client side scripts that want access to the session id. Using these techniques, cookies cannot be accessed by XSS and it won't be as easy as we showed to get a session id from a cookie manager.

The second step is to add a token to every request. Similar to the way we dealt with repeat forms in previous sections, we add a hidden field that contains a token. When a request is sent to the server, we can verify this token to prove that the request is unique.

```go
h := md5.New()
salt:="secret%^7&8888"
io.WriteString(h,salt+time.Now().String())
token:=fmt.Sprintf("%x",h.Sum(nil))
if r.Form["token"]!=token{
    // ask to log in
}
session.Set("token",token)

```


**Session id timeout**

Another solution is to add a create time for every session, and to replace expired session ids with new ones. This can prevent session hijacking under certain circumstances.

```go

createtime := session.Get("createtime")
if createtime == nil {
    session.Set("createtime", time.Now().Unix())
} else if (createtime.(int64) + 60) < (time.Now().Unix()) {
    sess.Destroy(c)
    session = sess.Start(c)
}
```

We set a value to save the create time and check if it's expired (I set 60 seconds here). This step can often thwart session hijacking attempts.

Combine the two solutions above and you will be able to prevent most session hijacking attempts from succeeding. On the one hand, session ids that are frequently reset will result in an attacker always getting expired and useless session ids; on the other hand, by setting the httponly property on cookies and ensuring that session ids can only be passed via cookies, all URL based attacks are mitigated.


This [section](https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/06.4.html) was very helpful for me to make this package
