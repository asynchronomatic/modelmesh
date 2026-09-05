package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"modelmesh/api"
)

func out() *os.File {
	return os.Stdout
}

func main() {
	if err := newCommand().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:                  "admincli",
		Usage:                 "ModelMesh admin and node CLI",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Usage:   "admin HTTP address",
				Value:   "http://127.0.0.1:4002",
				Sources: cli.EnvVars("MODELMESH_ADMIN_ADDR"),
			},
			&cli.StringFlag{
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "admin bearer token",
				Sources: cli.EnvVars("MODELMESH_ADMIN_TOKEN"),
			},
			&cli.StringFlag{
				Name:    "mesh",
				Aliases: []string{"m"},
				Usage:   "mesh id",
				Value:   "default",
			},
		},
		Commands: []*cli.Command{
			nodeCommand(),
			adminCommand(),
			{
				Name:      "redeem",
				Usage:     "Redeem an invite link (public, no admin token)",
				ArgsUsage: "INVITE_URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "node peer id", Required: true},
					&cli.StringFlag{Name: "name", Usage: "node name"},
				},
				Action: redeemInvite,
			},
			{
				Name:   "meshes",
				Usage:  "List meshes this token can access",
				Before: requireToken,
				Action: listMeshes,
			},
		},
	}
}

func requireToken(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if strings.TrimSpace(cmd.String("token")) == "" {
		return ctx, fmt.Errorf("--token is required (or set MODELMESH_ADMIN_TOKEN)")
	}
	return ctx, nil
}

func newAPI(cmd *cli.Command) *api.Client {
	addr := strings.TrimRight(cmd.String("addr"), "/")
	return api.NewClient(addr, cmd.String("token"))
}

func meshClient(cmd *cli.Command) (*api.MeshClient, error) {
	return newAPI(cmd).Mesh(cmd.String("mesh"))
}

func requireArg(cmd *cli.Command, name string) (string, error) {
	v := strings.TrimSpace(cmd.Args().First())
	if v == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return v, nil
}
