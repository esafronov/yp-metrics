// Package access includes middleware for checking ip in request from trusted subnet
package access

import (
	"net"
	"net/http"
)

const HeaderIp string = "X-Real-IP"

// ValidateIp server middleware for checking ip in request from trusted subnet
func ValidateIp(trustedSubnet string) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//if trustedSubnet is not empty, we make validation
			if trustedSubnet != "" {
				ip := r.Header.Get(HeaderIp)
				//cidr example 127.0.0.0/24
				_, ipv4Net, err := net.ParseCIDR(trustedSubnet)
				if err != nil {
					panic(err)
				}
				ipv4 := net.ParseIP(ip)
				//if ip is invalid
				if ipv4 == nil {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				//check network contains ip address
				if !ipv4Net.Contains(ipv4) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			h.ServeHTTP(w, r)
		})
	}
}
