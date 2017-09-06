package kickback

import (
	reactor "github.com/draganm/go-reactor"
	"github.com/draganm/immersadb"
	"github.com/draganm/immersadb/dbpath"
	"github.com/draganm/immersadb/modifier"
	"github.com/urfave/negroni"
)

type Context struct {
	ScreenContext   reactor.ScreenContext
	DB              *immersadb.ImmersaDB
	MountFunc       func()
	OnUserEventFunc func(*reactor.UserEvent)
	UnmountFunc     func()
	listeners       []*listener
}

type listener struct {
	db   *immersadb.ImmersaDB
	path dbpath.Path
	f    func(er modifier.EntityReader)
}

func (l *listener) unsubscribe() {
	l.db.RemoveListener(l.path, l)
}

func (l *listener) OnChange(r modifier.EntityReader) {
	l.f(r)
}

func (c *Context) Listen(dbpth dbpath.Path, f func(er modifier.EntityReader)) {
	l := &listener{db: c.DB, path: dbpth, f: f}
	c.listeners = append(c.listeners, l)
	c.DB.AddListener(dbpth, l)
}

func (c *Context) Mount() {
	if c.MountFunc != nil {
		c.MountFunc()
	}
}

func (c *Context) Unmount() {
	if c.UnmountFunc != nil {
		c.UnmountFunc()
	}
	for _, l := range c.listeners {
		l.unsubscribe()
	}
}

func (c *Context) OnUserEvent(evt *reactor.UserEvent) {
	if c.OnUserEventFunc != nil {
		c.OnUserEventFunc(evt)
	}
}

type kickback struct {
	reactor *reactor.Reactor
	db      *immersadb.ImmersaDB
}

func Run(addr string, db *immersadb.ImmersaDB, handlers []negroni.Handler) {
	k := &kickback{
		reactor: reactor.New(handlers...),
		db:      db,
	}
	for _, s := range Screens {
		k.AddScreen(s.Path, s.Factory)
	}
	k.reactor.Serve(addr)
}

func (k *kickback) AddScreen(path string, s func(*Context)) {
	k.reactor.AddScreen(path, func(ctx reactor.ScreenContext) reactor.Screen {
		sctx := &Context{
			ScreenContext: ctx,
			DB:            k.db,
		}
		s(sctx)
		return sctx
	})
}

type AddedScreen struct {
	Path    string
	Factory func(*Context)
}

var Screens = []*AddedScreen{}

func AddScreen(path string, s func(*Context)) {
	Screens = append(Screens, &AddedScreen{Path: path, Factory: s})
}
