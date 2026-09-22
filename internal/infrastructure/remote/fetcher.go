package remote

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-api/internal/domain/port"
	domainvideo "go-api/internal/domain/video"
)

type Fetcher struct {
	client       *http.Client
	allowedHosts map[string]struct{}
	allowHTTP    bool
}

func NewFetcher(allowedHosts []string, allowHTTP bool) *Fetcher {
	hosts := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			hosts[host] = struct{}{}
		}
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, domainvideo.ErrRemoteURLForbidden
			}
			if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
				return nil, domainvideo.ErrRemoteURLForbidden
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &Fetcher{
		client: &http.Client{
			Timeout:   10 * time.Minute,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return validateURL(req.URL, hosts, allowHTTP)
			},
		},
		allowedHosts: hosts,
		allowHTTP:    allowHTTP,
	}
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*port.RemoteObject, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, domainvideo.ErrRemoteURLForbidden
	}
	if err := validateURL(parsed, f.allowedHosts, f.allowHTTP); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("remote video returned status %d", resp.StatusCode)
	}

	return &port.RemoteObject{
		Body:        resp.Body,
		Size:        resp.ContentLength,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func validateURL(parsed *url.URL, allowedHosts map[string]struct{}, allowHTTP bool) error {
	if parsed == nil || parsed.Host == "" {
		return domainvideo.ErrRemoteURLForbidden
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && !(allowHTTP && scheme == "http") {
		return domainvideo.ErrRemoteURLForbidden
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return domainvideo.ErrRemoteURLForbidden
	}
	if len(allowedHosts) > 0 {
		if _, ok := allowedHosts[host]; !ok {
			return domainvideo.ErrRemoteURLForbidden
		}
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return domainvideo.ErrRemoteURLForbidden
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return domainvideo.ErrRemoteURLForbidden
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 {
		return true
	}
	return false
}
