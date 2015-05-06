package main

import (
	"github.com/etix/geoip"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var (
	geo *geoip.GeoIP
)

type result struct {
	ReqHostname string
	ClientIP    string
	ClientFR    bool
	ServerIP    string
	ServerFR    bool
	AnyFR       bool
}

func getIP(remoteAddr string) string {
	r := strings.Split(remoteAddr, ":")
	return r[0]
}

func lookupIP(host string) (string, error) {
	addr, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	for _, a := range addr {
		// Exclude ipv6 since we do not use the GeoIP Lite with v6 support
		if !strings.Contains(a, ":") {
			return a, nil
		}
	}
	return addr[0], nil
}

func resultPage(r render.Render, req *http.Request, u *url.URL) {
	var err error
	data := &result{}

	data.ReqHostname = u.Host
	data.ServerIP, err = lookupIP(u.Host)

	// Use the forwarded IP if any
	ip := req.Header.Get("X-Real-IP")
	if ip == "" {
		ip = req.RemoteAddr
	}

	data.ClientIP = getIP(ip)

	if err != nil {
		r.HTML(200, "index", "Adresse invalide")
		return
	}

	if country, _ := geo.GetCountry(data.ClientIP); country == "FR" {
		data.ClientFR = true
	}

	if country, _ := geo.GetCountry(data.ServerIP); country == "FR" {
		data.ServerFR = true
	}

	data.AnyFR = data.ClientFR || data.ServerFR

	r.HTML(200, "result", data)
}

func main() {
	var err error

	geo, err = geoip.Open()
	if err != nil {
		log.Fatal(err)
	}

	m := martini.Classic()
	m.Use(render.Renderer())
	m.Use(martini.Static("static"))

	m.Get("/", func(r render.Render, req *http.Request) {
		query := req.URL.Query()
		q := query.Get("q")

		if q != "" && !strings.HasPrefix(q, "http://") && !strings.HasPrefix(q, "https://") {
			q = "http://" + q
		}

		u, err := url.Parse(q)

		if err != nil || q == "" {
			r.HTML(200, "index", nil)
		} else {
			resultPage(r, req, u)
		}

	})

	m.RunOnAddr("127.0.0.1:8080")
}
