package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPServersRejectEmptyList(t *testing.T) {
	require.ErrorContains(t, runHTTPServers(context.Background(), time.Second), "at least one")
}

func TestHTTPServersDrainActiveRequests(t *testing.T) {
	for _, forceTimeout := range []bool{false, true} {
		name := "graceful"
		if forceTimeout {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			release := make(chan struct{})
			defer close(release)
			stopping := make(chan struct{}, 2)
			first, firstAddress := blockingHTTPServer(release, stopping)
			second, secondAddress := blockingHTTPServer(release, stopping)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			finished := make(chan error, 1)
			go func() { finished <- runHTTPServers(ctx, 200*time.Millisecond, first, second) }()
			firstRequest := startBlockingRequest(t, await(t, firstAddress))
			secondRequest := startBlockingRequest(t, await(t, secondAddress))
			cancel()
			for range 2 {
				await(t, stopping)
			}
			if !forceTimeout {
				select {
				case err := <-finished:
					t.Fatalf("shutdown returned before active requests finished: %v", err)
				default:
				}
				release <- struct{}{}
				release <- struct{}{}
			}
			shutdownErr := await(t, finished)
			firstErr, secondErr := await(t, firstRequest), await(t, secondRequest)
			if forceTimeout {
				require.ErrorIs(t, shutdownErr, context.DeadlineExceeded)
			} else {
				require.NoError(t, shutdownErr)
				require.NoError(t, firstErr)
				require.NoError(t, secondErr)
			}
		})
	}
}

func blockingHTTPServer(release, stopping chan struct{}) (*http.Server, <-chan string) {
	address := make(chan string, 1)
	server := &http.Server{
		Addr: "127.0.0.1:0",
		BaseContext: func(listener net.Listener) context.Context {
			address <- listener.Addr().String()
			return context.Background()
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = http.NewResponseController(w).Flush()
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}),
	}
	server.RegisterOnShutdown(func() { stopping <- struct{}{} })
	return server, address
}

func startBlockingRequest(t *testing.T, address string) <-chan error {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	response, err := client.Get("http://" + address)
	require.NoError(t, err)
	finished := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		finished <- err
	}()
	return finished
}

func await[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	select {
	case result := <-channel:
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP test did not make progress within its deadline")
		var zero T
		return zero
	}
}

func TestHTTPServersCloseEarlierListenerOnBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	err = runHTTPServers(context.Background(), time.Second, &http.Server{Addr: address}, &http.Server{Addr: address})
	require.ErrorContains(t, err, "listen on "+address)
	listener, err = net.Listen("tcp", address)
	require.NoError(t, err, "first listener must be released after the second one fails")
	require.NoError(t, listener.Close())
}

func TestHTTPServersStopBothOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	first := &http.Server{Addr: "127.0.0.1:0"}
	second := &http.Server{Addr: "127.0.0.1:0"}
	stopped := make(chan struct{}, 2)
	first.RegisterOnShutdown(func() { stopped <- struct{}{} })
	second.RegisterOnShutdown(func() { stopped <- struct{}{} })
	cancel()
	require.NoError(t, runHTTPServers(ctx, time.Second, first, second))
	for range 2 {
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("one HTTP server was not shut down")
		}
	}
}
