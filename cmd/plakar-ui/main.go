// Command plakar-ui sert le tableau de bord multi-Stores de Plakar.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plakar-ui/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a, err := app.Start(ctx, app.Options{
		Getenv:   os.Getenv,
		Now:      time.Now,
		After:    time.After,
		Location: time.Local,
		Log:      os.Stderr,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "plakar-ui :", err)
		os.Exit(1)
	}
	<-ctx.Done()
	a.Close()
}
