package cli

import (
	"fmt"
	"os"
	"strings"

	"zerodha-connect/internal/config"
	"zerodha-connect/internal/kite"
	"zerodha-connect/internal/logger"
	"zerodha-connect/internal/ui"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// profileCmd represents the profile command
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Fetch and display user profile details",
	Long: `Fetch user profile information from Zerodha Kite API and display it in a well-formatted table.

This command fetches comprehensive user profile information including:
- User ID and personal details
- Enabled exchanges and products
- Order types available
- User meta information

API credentials must be provided via config file or command line flags.

Examples:
  # Fetch profile using config file
  zerodha-connect profile

  # Fetch profile with specific config file
  zerodha-connect profile --config my-config.yaml

  # Enable verbose logging
  zerodha-connect profile --verbose`,
	RunE: runProfile,
}

func runProfile(cmd *cobra.Command, args []string) error {
	// Load configuration
	conf, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %v", configFile, err)
	}

	// Validate API credentials
	if conf.APIKey == "" || conf.APISecret == "" {
		return fmt.Errorf("API credentials required. Please provide them in the config file:\n" +
			"  • api_key: Your Zerodha API key\n" +
			"  • api_secret: Your Zerodha API secret")
	}

	// Initialize logger
	appLogger := logger.NewSilent()
	if verbose {
		appLogger = logger.New("profile.log")
		appLogger.Println("🔧 Verbose mode enabled")
	}

	fmt.Println("👤 Fetching user profile...")

	// Initialize Kite client with the config file path
	kiteClient := kite.NewClientWithConfigPath(conf, appLogger, configFile)

	// Use the new authentication method with token validation
	err = kiteClient.AuthenticateWithTokenValidation()
	if err != nil {
		// Check if it's an authentication error with expired token
		if authErr, ok := err.(*kite.AuthenticationError); ok && authErr.Type == kite.AuthErrorTokenExpired {
			// Ask user if they want to start auth process
			if !ui.ConfirmAuthRestart() {
				fmt.Println("❌ Authentication cancelled by user")
				return nil
			}

			// Clear the expired token and start fresh auth flow
			conf.RequestToken = ""
			err = kiteClient.Authenticate()
			if err != nil {
				return fmt.Errorf("authentication failed: %v", err)
			}
		} else {
			return fmt.Errorf("authentication failed: %v", err)
		}
	}

	// Fetch user profile
	profile, err := kiteClient.GetUserProfile()
	if err != nil {
		return fmt.Errorf("failed to fetch profile: %v", err)
	}

	// Display profile in table format
	displayProfile(profile)

	return nil
}

func displayProfile(profile *kiteconnect.UserProfile) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("👤 USER PROFILE")
	fmt.Println(strings.Repeat("=", 60))

	// Basic Information Table
	fmt.Println("\n📋 Basic Information:")
	basicTable := tablewriter.NewWriter(os.Stdout)
	basicTable.SetHeader([]string{"Field", "Value"})
	basicTable.SetBorder(true)
	basicTable.SetRowLine(true)
	basicTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
	)
	basicTable.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Normal},
	)

	basicData := [][]string{
		{"User ID", profile.UserID},
		{"User Name", profile.UserName},
		{"Short Name", profile.UserShortName},
		{"Email", profile.Email},
		{"User Type", profile.UserType},
		{"Broker", profile.Broker},
	}

	for _, row := range basicData {
		basicTable.Append(row)
	}
	basicTable.Render()

	// Trading Information Table
	fmt.Println("\n💼 Trading Information:")
	tradingTable := tablewriter.NewWriter(os.Stdout)
	tradingTable.SetHeader([]string{"Category", "Available Options"})
	tradingTable.SetBorder(true)
	tradingTable.SetRowLine(true)
	tradingTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
	)
	tradingTable.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Normal},
	)

	tradingData := [][]string{
		{"Exchanges", strings.Join(profile.Exchanges, ", ")},
		{"Products", strings.Join(profile.Products, ", ")},
		{"Order Types", strings.Join(profile.OrderTypes, ", ")},
	}

	for _, row := range tradingData {
		tradingTable.Append(row)
	}
	tradingTable.Render()

	// Meta Information Table (if available)
	if profile.Meta.DematConsent != "" {
		fmt.Println("\n🔧 Meta Information:")
		metaTable := tablewriter.NewWriter(os.Stdout)
		metaTable.SetHeader([]string{"Field", "Value"})
		metaTable.SetBorder(true)
		metaTable.SetRowLine(true)
		metaTable.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiYellowColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiYellowColor},
		)
		metaTable.SetColumnColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor},
			tablewriter.Colors{tablewriter.Normal},
		)

		metaData := [][]string{
			{"Demat Consent", profile.Meta.DematConsent},
		}

		for _, row := range metaData {
			metaTable.Append(row)
		}
		metaTable.Render()
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("✅ Profile fetched successfully!")
}
