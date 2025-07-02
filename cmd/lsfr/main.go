package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/st3v3nmw/lsfr/internal/registry"
	"github.com/urfave/cli/v3"

	_ "github.com/st3v3nmw/lsfr/challenges"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "test",
				Usage: "<Test>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fn, err := registry.GetStage("key-value-store", 1)
					if err != nil {
						return err
					}

					s := fn()
					r := s.Run(context.Background(), "./little-key-value")
					fmt.Printf("%#v\n", r)
					fmt.Println(r)

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
