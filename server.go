package api

import (
  "bytes"
  "io"
  "log"
  "net/http"

  "github.com/badgerodon/s"
  "code.google.com/p/go.net/websocket"
)

type (
  HandlerFunc func(Listener,s.Expression) interface{}
  Handler func(Listener,s.List)
  result struct {
    id uint64
    value interface{}
  }
)

func (this result) EncodeS() (exp s.Expression, err error) {
  var v1, v2, v3 s.Expression
  v1 = s.Identifier("result")
  v2, err = s.Encode(this.id)
  if err != nil {
    return
  }
  v3, err = s.Encode(this.value)
  if err != nil {
    return
  }
  exp = s.NewList(v1, v2, v3)
  return
}

var (
  handlers = map[string]Handler{}
)

func WebSocketHandler(conn *websocket.Conn) {
  // Make a new listener which we'll close when the websocket is closed
  listener := NewListener()
  defer listener.Close()

  // Read messages from the user
  go func() {
    for {
      exp, err := s.Read(conn)
      if err != nil {
        if err != io.EOF {
          log.Printf("<- [ERROR] %v", err)
        }
        close(listener)
        return
      }
      log.Printf("<- %v", exp)

      lst, ok := exp.(s.List)
      if !ok {
        listener.Errorf("Invalid request: expected List")
        continue
      }

      n, err := lst.Head()
      if err != nil {
        listener.Errorf("Invalid request: expected at least one item")
        continue
      }

      var action string
      err = n.Scan(&action)
      if err != nil {
        listener.Errorf("Invalid request: expected string")
        continue
      }

      h, ok := handlers[action]
      if !ok {
        listener.Errorf("Unknown action: %v", action)
        continue
      }

      tail, _ := lst.Tail()
      h(listener, tail)
    }
  }()

  // Write messages from the server
  for {
    val, ok := <- listener
    if !ok {
      break
    }
    exp, err := s.Encode(val)
    if err != nil {
      exp, _ = s.Encode(err)
    }
    var buf bytes.Buffer
    err = exp.Write(&buf)
    if err != nil {
      break
    }
    str := buf.String()
    log.Printf("-> %v", str)
    _, err = io.WriteString(conn, str)
    if err != nil {
      break
    }
  }
}

func HandleFunc(name string, handler HandlerFunc) {
  handlers[name] = func(listener Listener, args s.List) {
    if len(args) != 2 {
      listener.Errorf("Expected request id & arguments")
      return
    }
    var id uint64
    err := args[0].Scan(&id)
    if err != nil {
      listener.Errorf("Expected request id to be a number got %v", args[1])
      return
    }
    val := handler(listener, args[1])
    listener <- result{id,val}
  }
}

func init() {
  http.Handle("/ws", websocket.Handler(WebSocketHandler))
}
