package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}

	p := newParser()

	wg.Add(1)
	go p.Run(wg)

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.File("index.html")
	})

	e.GET("/sse", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")

		var filter Filter

		if err := c.Bind(&filter); err != nil {
			return err
		}

		filteredEntries := make(map[string]*Entry)

		for key, entry := range p.Entries {
			if filter.isTrue(*entry) {
				filteredEntries[key] = entry
			}
		}

		if err := writeFlush(c, Initial, filteredEntries); err != nil {
			return err
		}

		for {
			select {
			case <-c.Request().Context().Done():
				return nil
			case event := <-p.EventBus:
				if !filter.isTrue(*event.Entry) {
					continue
				}

				if err := writeFlush(c, event.Type, event.Entry); err != nil {
					return err
				}

				break
			}
		}
	})

	e.GET("/vue.global.js", func(c echo.Context) error {
		return c.File("vue.global.js")
	})

	go e.Start(":1323")

	wg.Wait()
}

type Filter struct {
	Server string
	Side string
}

func (f Filter) isTrue(entry Entry) bool {
	return f.Server == entry.Server && f.Side == entry.Side
}

func writeFlush(c echo.Context, event EventType, obj interface{}) error {
	message := packMessage(event, obj)
	_, err := c.Response().Write(message)
	if err != nil {
		return err
	}

	c.Response().Flush()
	return nil
}

func packMessage(event EventType, obj interface{}) []byte {
	evt := fmt.Sprintf("event: %s\n", event)
	json, _ := json.Marshal(obj)
	text := fmt.Sprintf("data: %s\n\n", json)

	return []byte(evt + text)
}