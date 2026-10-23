package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
