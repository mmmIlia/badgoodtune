package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

func runHTTPServers(ctx context.Context, timeout time.Duration, servers ...*http.Server) error {
	if len(servers) == 0 {
		return errors.New("at least one HTTP server is required")
	}
	listeners := make([]net.Listener, 0, len(servers))
	for _, server := range servers {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return fmt.Errorf("listen on %s: %w", server.Addr, err)
		}
		defer listener.Close()
		defer server.Close()
		listeners = append(listeners, listener)
	}

	results := make(chan error, len(servers))
	for i, server := range servers {
		go func() {
			err := server.Serve(listeners[i])
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			} else if err != nil {
				err = fmt.Errorf("serve %s: %w", server.Addr, err)
			}
			results <- err
		}()
	}

	var problems []error
	remaining := len(servers)
	select {
	case err := <-results:
		problems = append(problems, err)
		remaining--
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	shutdownResults := make(chan error, len(servers))
	for _, server := range servers {
		go func() {
			err := server.Shutdown(shutdownContext)
			if err != nil {
				_ = server.Close()
				err = fmt.Errorf("shutdown %s: %w", server.Addr, err)
			}
			shutdownResults <- err
		}()
	}
	for range servers {
		problems = append(problems, <-shutdownResults)
	}
	for range remaining {
		problems = append(problems, <-results)
	}
	return errors.Join(problems...)
}
