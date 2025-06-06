package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenBrowser opens the specified URL in the user's default browser.
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// GetRequestToken prompts the user for the request token after they log in.
func GetRequestToken(loginURL string) (string, error) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔑 AUTHENTICATION REQUIRED")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Please open this URL in your browser: %s\n", loginURL)
	fmt.Println("1. Login to Zerodha")
	fmt.Println("2. After successful login, you'll be redirected to a URL")
	fmt.Println("3. Copy the 'request_token' parameter from the redirected URL")
	fmt.Println("4. Paste it below and press Enter")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Print("Enter request token: ")

	reader := bufio.NewReader(os.Stdin)
	requestToken, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read request token: %v", err)
	}
	requestToken = strings.TrimSpace(requestToken)

	if requestToken == "" {
		return "", fmt.Errorf("request token cannot be empty")
	}

	return requestToken, nil
}

// FetchPlan holds the details for the data fetching operation to be confirmed by the user.
type FetchPlan struct {
	ValidInstruments          int
	FromDate                  string
	ToDate                    string
	Interval                  string
	RateLimitPerSecond        int
	ChunkExplanation          string
	ChunkSizeInfo             string
	InstrumentsPerRequest     int
	TotalAPICalls             int
	EstimatedMinutes          int
	EstimatedRemainingSeconds int
}

// ConfirmExecution displays the fetching plan and asks for user confirmation.
func ConfirmExecution(plan FetchPlan) bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📈 DATA FETCHING PLAN")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("🎯 Valid instruments: %d\n", plan.ValidInstruments)
	fmt.Printf("📅 Date range: %s to %s\n", plan.FromDate, plan.ToDate)
	fmt.Printf("⏱️  Interval: %s\n", plan.Interval)
	fmt.Println()
	fmt.Println("🧩 CHUNKING STRATEGY:")
	fmt.Printf("  • API Rate Limit: %d requests/second globally\n", plan.RateLimitPerSecond)
	fmt.Printf("  • Window Limit: %s\n", plan.ChunkExplanation)
	fmt.Printf("  • Chunk size: %s\n", plan.ChunkSizeInfo)
	fmt.Printf("  • Instrument limit: %d per request\n", plan.InstrumentsPerRequest)
	fmt.Printf("  • Result: %d total chunks across all instruments\n", plan.TotalAPICalls)
	fmt.Println()
	fmt.Printf("📡 Total API calls needed: %d\n", plan.TotalAPICalls)
	if plan.EstimatedMinutes > 0 {
		fmt.Printf("⏳ Estimated time: ~%d minutes %d seconds\n", plan.EstimatedMinutes, plan.EstimatedRemainingSeconds)
	} else {
		fmt.Printf("⏳ Estimated time: ~%d seconds\n", plan.EstimatedRemainingSeconds)
	}
	fmt.Println(strings.Repeat("=", 60))
	fmt.Print("Do you want to proceed? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read user input: %v\n", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes"
}

// ConfirmAuthRestart asks the user if they want to restart the authentication process
func ConfirmAuthRestart() bool {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🚨 AUTHENTICATION PROBLEM")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("❌ Your request token appears to be expired or invalid.")
	fmt.Println("🔑 API key and secret are present in your configuration.")
	fmt.Println("")
	fmt.Println("To proceed, you need to start the authentication process again.")
	fmt.Println("This will:")
	fmt.Println("  • Open your browser for Zerodha login")
	fmt.Println("  • Generate a new request token")
	fmt.Println("  • Save the new token to your config file")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Print("Do you want to start the authentication process? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read user input: %v\n", err)
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes"
}
