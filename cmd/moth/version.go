package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/aloisdeniel/moth/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the moth version",
		Run: func(*cobra.Command, []string) {
			fmt.Printf("moth %s (%s/%s)\n", version.Version, runtime.GOOS, runtime.GOARCH)
		},
	}
}
