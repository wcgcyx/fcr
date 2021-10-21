package daemon

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/wcgcyx/fcr/internal/api"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/node"
)

// Daemon starts a daemon from path.
// It takes a path as the argument.
// It returns error.
func Daemon(parentCtx context.Context, path string) error {
	// First check if given path exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path %v does not exists", path)
	}
	// Load the config file
	conf, err := config.ReadFromPath(path)
	if err != nil {
		return fmt.Errorf("error loading config file: %v", err.Error())
	}
	// Context
	ctx, cancel := context.WithCancel(parentCtx)
	// Start the node
	node, err := node.NewNode(ctx, conf)
	if err != nil {
		cancel()
		return fmt.Errorf("error starting daemon: %v", err.Error())
	}
	// Start API Server
	err = api.NewServer(ctx, conf, node)
	if err != nil {
		cancel()
		return fmt.Errorf("error starting API server: %v", err.Error())
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		// Handle ctrl + c
		<-c
		cancel()
	}()
	for {
		// Loop forever
	}
}
