package api

import (
  "errors"
  "fmt"
  "sync"
)

type (
  Listener chan interface{}
  Void struct {}
)

var (
  db = struct {
    listeners map[Listener][]string
    rooms map[string][]Listener
    lock sync.Mutex
  } {
    listeners: map[Listener][]string{},
    rooms: map[string][]Listener{},
  }
)

func NewListener() Listener {
  listener := Listener(make(chan interface{}))
  db.lock.Lock()
  db.listeners[listener] = []string{}
  db.lock.Unlock()
  return listener
}
func (this Listener) Enter(room string) {
  db.lock.Lock()
  defer db.lock.Unlock()

  listeners, ok := db.rooms[room]
  if !ok {
    listeners = []Listener{this}
  } else {
    listeners = append(listeners, this)
  }
  db.rooms[room] = listeners
}
func (this Listener) Leave(room string) {
  db.lock.Lock()
  defer db.lock.Unlock()

  listeners, ok := db.rooms[room]
  if ok {
    nls := make([]Listener, 0, len(listeners) - 1)
    for _, listener := range listeners {
      if listener != this {
        nls = append(nls, listener)
      }
    }
    if len(nls) > 0 {
      db.rooms[room] = nls
    } else {
      delete(db.rooms, room)
    }
  }
}
func (this Listener) Close() {
  db.lock.Lock()
  rooms, ok := db.listeners[this]
  if ok {
    delete(db.listeners, this)
  }
  db.lock.Unlock()

  if ok {
    for _, room := range rooms {
      this.Leave(room)
    }
  }
}
func (this Listener) Errorf(str string, values ... interface{}) {
  this<- errors.New(fmt.Sprintf(str, values...))
}
func (this Listener) Send(value interface{}) {
  this<- value
}