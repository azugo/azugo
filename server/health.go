package server

import (
	"fmt"
	"os"

	"azugo.io/core/http"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// HealthCommand returns a cobra command that checks whether the HTTP server is responding.
// The healthzPath argument sets the health check endpoint path (e.g. "/healthz").
//
//	cli.Register(server.HealthCommand("/healthz", server.Options{Configuration: &myConf}))
func HealthCommand(healthzPath string, opt Options) *cobra.Command {
	return &cobra.Command{
		Use:           "health",
		Short:         "Check health of the server",
		Long:          `Check if the web server is running and responding to healthz request`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newApp(cmd, opt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
				os.Exit(1)

				return nil
			}

			conf := a.Config().Server
			client := a.HTTPClient()

			scheme := "http"
			addr := conf.HTTP.Address
			port := conf.HTTP.Port

			if !conf.HTTP.Enabled {
				scheme = "https"
				addr = conf.HTTPS.Address
				port = conf.HTTPS.Port
				client = client.WithOptions(&http.TLSConfig{InsecureSkipVerify: true}) //nolint:gosec
			}

			if addr == "" || addr == "0.0.0.0" {
				addr = "localhost"
			}

			url := fmt.Sprintf("%s://%s:%d%s", scheme, addr, port, healthzPath)
			if _, err = client.Get(url); err != nil {
				a.Log().Error("server health check failed", zap.Error(err))
				os.Exit(1)

				return nil
			}

			return nil
		},
	}
}
