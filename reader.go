package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"launchpad.net/xmlpath"
)

type reader struct {
	url  string
	out  chan<- datapoint
	intv time.Duration

	stop chan struct{}
	lock sync.Mutex
}

func (r *reader) Serve() {
	r.lock.Lock()
	r.stop = make(chan struct{})
	r.lock.Unlock()

	ticker := time.NewTicker(r.intv)
	defer ticker.Stop()

	log.Println(r, "starting")
	defer log.Println(r, "exiting")

	r.out <- parseURL(r.url)
	for {
		select {
		case <-ticker.C:
			r.out <- parseURL(r.url)
		case <-r.stop:
			return
		}
	}
}

func (r *reader) Stop() {
	r.lock.Lock()
	close(r.stop)
	r.lock.Unlock()
}

func (r *reader) String() string {
	return fmt.Sprintf("reader@%p", r)
}

func parseURL(url string) datapoint {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	return parseXML(resp.Body)
}

func parseXML(fd io.Reader) datapoint {
	root, err := xmlpath.Parse(fd)
	if err != nil {
		log.Fatal(err)
	}

	result := datapoint{
		time: time.Now(),
	}
	families := xmlpath.MustCompile("/Devices-Detail-Response/*[Name]")
	namePath := xmlpath.MustCompile("Name")
	iter := families.Iter(root)

	for iter.Next() {
		name, ok := namePath.String(iter.Node())
		if !ok {
			continue
		}
		switch name {
		case "DS18B20":
			// Thermometer
			strVal, _ := xmlpath.MustCompile("Temperature").String(iter.Node())
			result.temperature, _ = strconv.ParseFloat(strVal, 64)

		case "DS2423":
			// Counter
			strVal, _ := xmlpath.MustCompile("Counter_A").String(iter.Node())
			result.wattHours, _ = strconv.ParseInt(strVal, 10, 64)
		}
	}

	return result
}
