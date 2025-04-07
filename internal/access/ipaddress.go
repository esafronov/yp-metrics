// Package access includes middleware for checking ip in request from trusted subnet
package access

import (
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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

// UnaryValidateIpInterceptor is the interceptor for gRPC server for validation ip address from trusted subnet
func UnaryValidateIpInterceptor(trustedSubnet string) func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if trustedSubnet != "" {
			p, _ := peer.FromContext(ctx)
			_, ipv4Net, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				panic(err)
			}
			tcpAddr, ok := p.Addr.(*net.TCPAddr)
			if !ok {
				return nil, status.Error(codes.Unauthenticated, "invalid ip")
			}
			ipv4 := tcpAddr.IP
			//if ip is invalid
			if ipv4 == nil {
				return nil, status.Error(codes.Unauthenticated, "invalid ip")
			}
			//check network contains ip address
			if !ipv4Net.Contains(ipv4) {
				return nil, status.Error(codes.Unauthenticated, "invalid ip")
			}
		}
		return handler(ctx, req)
	}
}
