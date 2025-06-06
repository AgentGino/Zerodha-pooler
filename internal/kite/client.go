package kite

import (
	"context"
	"fmt"
	"log"
	"time"

	"zerodha-connect/internal/config"
	"zerodha-connect/internal/ui"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"golang.org/x/time/rate"
)

const (
	// RateLimitRequestsPerSecond is the number of API requests allowed per second.
	RateLimitRequestsPerSecond = 3
	// RateLimitBurst is the burst allowance for the rate limiter.
	RateLimitBurst = 1
)

// Client is a wrapper around the Kite Connect client.
type Client struct {
	kc         *kiteconnect.Client
	limiter    *rate.Limiter
	logger     *log.Logger
	conf       *config.Config
	configPath string
}

// NewClient creates a new Kite client.
func NewClient(conf *config.Config, logger *log.Logger) *Client {
	kc := kiteconnect.New(conf.APIKey)
	limiter := rate.NewLimiter(RateLimitRequestsPerSecond, RateLimitBurst)
	return &Client{
		kc:      kc,
		limiter: limiter,
		logger:  logger,
		conf:    conf,
	}
}

// NewClientWithConfigPath creates a new Kite client with config file path.
func NewClientWithConfigPath(conf *config.Config, logger *log.Logger, configPath string) *Client {
	kc := kiteconnect.New(conf.APIKey)
	limiter := rate.NewLimiter(RateLimitRequestsPerSecond, RateLimitBurst)
	return &Client{
		kc:         kc,
		limiter:    limiter,
		logger:     logger,
		conf:       conf,
		configPath: configPath,
	}
}

// getConfigPath returns the config path to save to, defaulting to config.yaml if not set
func (c *Client) getConfigPath() string {
	if c.configPath != "" {
		return c.configPath
	}
	return "config.yaml"
}

// Authenticate handles the full authentication flow.
func (c *Client) Authenticate() error {
	if c.conf.RequestToken != "" {
		c.logger.Println("✅ Request token found in config. Proceeding...")
		c.kc.SetAccessToken(c.conf.RequestToken)
		return nil
	}

	c.logger.Println("🔐 No request token found. Starting authentication flow...")

	if c.conf.APIKey == "" || c.conf.APISecret == "" {
		return fmt.Errorf("API key and API secret are required for authentication")
	}

	loginURL := c.kc.GetLoginURL()
	c.logger.Printf("🌐 Opening browser for Zerodha login...")

	if err := ui.OpenBrowser(loginURL); err != nil {
		c.logger.Printf("⚠️  Failed to open browser automatically: %v", err)
	}

	requestToken, err := ui.GetRequestToken(loginURL)
	if err != nil {
		return err
	}

	c.logger.Printf("🔄 Exchanging request token for access token...")

	data, err := c.kc.GenerateSession(requestToken, c.conf.APISecret)
	if err != nil {
		return fmt.Errorf("failed to generate session: %v", err)
	}

	c.conf.RequestToken = data.AccessToken
	configPath := c.getConfigPath()
	if err := config.Save(configPath, c.conf); err != nil {
		return fmt.Errorf("failed to save request token to config: %v", err)
	}

	c.kc.SetAccessToken(c.conf.RequestToken)
	c.logger.Printf("✅ Authentication successful! Request token saved to %s", configPath)
	return nil
}

// AuthenticateWithTokenValidation handles authentication with proper token validation
func (c *Client) AuthenticateWithTokenValidation() error {
	if c.conf.RequestToken != "" {
		c.logger.Println("✅ Request token found in config. Validating...")
		c.kc.SetAccessToken(c.conf.RequestToken)

		// Test the token by making a simple API call
		if err := c.limiter.Wait(context.Background()); err != nil {
			return fmt.Errorf("rate limiter error: %v", err)
		}

		_, err := c.kc.GetUserProfile()
		if err != nil {
			// Request token is present but invalid/expired
			return &AuthenticationError{
				Type:    AuthErrorTokenExpired,
				Message: "Request token appears to be expired or invalid",
				Cause:   err,
			}
		}

		c.logger.Println("✅ Request token is valid")
		return nil
	}

	// No request token present - start auth flow
	return c.startAuthenticationFlow()
}

// startAuthenticationFlow handles the OAuth flow
func (c *Client) startAuthenticationFlow() error {
	c.logger.Println("🔐 No request token found. Starting authentication flow...")

	if c.conf.APIKey == "" || c.conf.APISecret == "" {
		return fmt.Errorf("API key and API secret are required for authentication")
	}

	loginURL := c.kc.GetLoginURL()
	c.logger.Printf("🌐 Opening browser for Zerodha login...")

	if err := ui.OpenBrowser(loginURL); err != nil {
		c.logger.Printf("⚠️  Failed to open browser automatically: %v", err)
	}

	requestToken, err := ui.GetRequestToken(loginURL)
	if err != nil {
		return err
	}

	c.logger.Printf("🔄 Exchanging request token for access token...")

	data, err := c.kc.GenerateSession(requestToken, c.conf.APISecret)
	if err != nil {
		return fmt.Errorf("failed to generate session: %v", err)
	}

	c.conf.RequestToken = data.AccessToken
	configPath := c.getConfigPath()
	if err := config.Save(configPath, c.conf); err != nil {
		return fmt.Errorf("failed to save request token to config: %v", err)
	}

	c.kc.SetAccessToken(c.conf.RequestToken)
	c.logger.Printf("✅ Authentication successful! Request token saved to %s", configPath)
	return nil
}

// AuthenticationError represents different types of authentication errors
type AuthenticationError struct {
	Type    AuthErrorType
	Message string
	Cause   error
}

func (e *AuthenticationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// AuthErrorType represents the type of authentication error
type AuthErrorType int

const (
	AuthErrorTokenExpired AuthErrorType = iota
	AuthErrorMissingCredentials
	AuthErrorAPIFailure
)

// GetKiteConnectClient returns the underlying Kite Connect client instance.
func (c *Client) GetKiteConnectClient() *kiteconnect.Client {
	return c.kc
}

// GetHistoricalData fetches historical data for a given instrument.
func (c *Client) GetHistoricalData(instrumentToken int, interval string, from, to time.Time) ([]kiteconnect.HistoricalData, error) {
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %v", err)
	}

	candles, err := c.kc.GetHistoricalData(instrumentToken, interval, from, to, false, false)
	if err != nil {
		return nil, fmt.Errorf("API error: %v", err)
	}
	return candles, nil
}

// GetUserProfile fetches the user profile information.
func (c *Client) GetUserProfile() (*kiteconnect.UserProfile, error) {
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %v", err)
	}

	profile, err := c.kc.GetUserProfile()
	if err != nil {
		return nil, fmt.Errorf("API error: %v", err)
	}
	return &profile, nil
}
