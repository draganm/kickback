package kickback

import (
	"log"

	reactor "github.com/draganm/go-reactor"
	"github.com/draganm/immersadb"
	"github.com/urfave/negroni"
)

type Context struct {
	ScreenContext reactor.ScreenContext
	DB            *immersadb.ImmersaDB
	MountFunc     func()
	UserEventFunc func(*reactor.UserEvent)
	UnmountFunc   func()
}

func (c *Context) Mount() {
	if c.MountFunc != nil {
		c.MountFunc()
		log.Println("c")
	}
}

func (c *Context) Unmount() {
	if c.UnmountFunc != nil {
		c.UnmountFunc()
	}
}

func (c *Context) OnUserEvent(evt *reactor.UserEvent) {
	if c.UserEventFunc != nil {
		c.UserEventFunc(evt)
	}
}

type Kickback struct {
	reactor *reactor.Reactor
	db      *immersadb.ImmersaDB
}

func New(db *immersadb.ImmersaDB, handlers ...negroni.Handler) *Kickback {
	return &Kickback{
		reactor: reactor.New(handlers...),
		db:      db,
	}
}

func (k *Kickback) AddScreen(path string, s func(*Context)) {
	k.reactor.AddScreen(path, func(ctx reactor.ScreenContext) reactor.Screen {
		sctx := &Context{
			ScreenContext: ctx,
			DB:            k.db,
		}
		s(sctx)
		return sctx
	})
}

func (k *Kickback) Serve(addr string) {
	k.reactor.Serve(addr)
}
