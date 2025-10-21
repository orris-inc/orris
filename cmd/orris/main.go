package main

import (
	"os"

	"github.com/spf13/cobra"

	"orris/internal/interfaces/cli/migrate"
	"orris/internal/interfaces/cli/server"
)

// @title Orris API
// @version 1.0
// @description A modern Go application with RESTful API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@orris.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @BasePath /

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	rootCmd := &cobra.Command{
		Use:   "orris",
		Short: "Orris - A modern Go application",
		Long:  `Orris is a production-ready Go application with built-in server, migration tools, and administrative commands.`,
	}

	rootCmd.AddCommand(
		server.NewCommand(),
		migrate.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
