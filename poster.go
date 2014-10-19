package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type post struct {
	Name    string          `json:"name"`
	Columns []string        `json:"columns"`
	Points  [][]interface{} `json:"points"`
}

type poster struct {
	url string
	in  <-chan datapoint

	stop chan struct{}
	lock sync.Mutex
}

func (p *poster) Serve() {
	p.lock.Lock()
	p.stop = make(chan struct{})
	p.lock.Unlock()

	log.Println(p, "starting")
	defer log.Println(p, "exiting")

	var buffer []datapoint
	for {
		select {
		case data, ok := <-p.in:
			if !ok {
				return
			}
			buffer = append(buffer, data)
			err := p.post(buffer)
			if err != nil {
				log.Println("post:", err, "(buffering)")
			} else {
				buffer = nil
			}

		case <-p.stop:
			return
		}
	}
}

func (p *poster) Stop() {
	p.lock.Lock()
	close(p.stop)
	p.lock.Unlock()
}

func (p *poster) String() string {
	return fmt.Sprintf("poster@%p", p)
}

func (p *poster) post(buffer []datapoint) error {
	points := post{
		Name:    "env",
		Columns: []string{"time", "temperature", "wattHours"},
	}
	for _, dp := range buffer {
		points.Points = append(points.Points, []interface{}{dp.time.Unix() * 1000, dp.temperature, dp.wattHours})
	}

	postData, err := json.Marshal([]post{points})
	if err != nil {
		return err
	}
	if debug {
		log.Printf("%s", postData)
	}

	resp, err := http.Post(p.url, "application/json", bytes.NewBuffer(postData))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	return nil
}
