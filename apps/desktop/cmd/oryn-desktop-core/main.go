package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/lunararch/oryn/apps/desktop/internal/launcher"
	"github.com/lunararch/oryn/apps/desktop/internal/serverapi"
)

func main() {
	server := flag.String("server", "", "Oryn server URL, e.g. http://pinas:8080")
	token := flag.String("token", os.Getenv("ORYN_CLIENT_TOKEN"), "paired client token")
	root := flag.String("root", "", "local library root")
	game := flag.String("game", "", "server-relative game path")
	executable := flag.String("exe", "", "candidate path relative to the game directory (optional)")
	var args string
	flag.StringVar(&args, "args", "", "comma-separated launch arguments")
	flag.Parse()

	if *server == "" || *token == "" || *root == "" || *game == "" {
		fmt.Fprintln(os.Stderr, "server, token, root, and game are required")
		os.Exit(2)
	}

	gameDir := *root + string(os.PathSeparator) + *game
	candidates, err := launcher.Detect(gameDir, runtime.GOOS)
	if err != nil {
		fmt.Fprintln(os.Stderr, "detect launch candidates:", err)
		os.Exit(1)
	}
	var candidate launcher.Candidate
	if *executable != "" {
		candidate = launcher.Candidate{RelativePath: *executable}
	} else if len(candidates) > 0 {
		candidate = candidates[0]
	} else {
		fmt.Fprintln(os.Stderr, "no supported launch candidate found")
		os.Exit(1)
	}

	var launchArgs []string
	if strings.TrimSpace(args) != "" {
		for _, arg := range strings.Split(args, ",") {
			launchArgs = append(launchArgs, strings.TrimSpace(arg))
		}
	}

	client := serverapi.New(*server)
	client.SetToken(*token)
	service := &launcher.Service{API: client}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("launching %s\n", candidate.RelativePath)
	result, err := service.LaunchAndTrack(ctx, *game, *root, *game, candidate, launchArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "launch failed:", err)
		os.Exit(1)
	}
	fmt.Printf("session %s finished with exit code %d\n", result.SessionID, result.ExitCode)
}
