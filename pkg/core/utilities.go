package core

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"modelmesh/pkg/log"
)

type Service interface {
	Serve(ctx context.Context) error
}

func RunInterruptible(services ...Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1) // buffer of 1 so you don't miss the signal
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
	}()

	var wg sync.WaitGroup

	for _, service := range services {
		wg.Go(func() {
			err := service.Serve(ctx)
			if err != nil {
				log.Errorf("service %T exited. Err:%v\n", service, err)
			}
			cancel()
		})
	}
	wg.Wait()
	return nil
}

func RunInterruptibleContext(ctx context.Context, services ...Service) error {
	var wg sync.WaitGroup

	for _, service := range services {
		wg.Go(func() {
			err := service.Serve(ctx)
			if err != nil {
				log.Errorf("Could not initialize admin server. Err:%v\n", err)
			}
		})

	}
	wg.Wait()
	return nil
}
