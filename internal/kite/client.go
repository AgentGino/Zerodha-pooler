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
	kc      *kiteconnect.Client
	limiter *rate.Limiter
	logger  *log.Logger
	conf    *config.Config
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

// Authenticate handles the full authentication flow.
func (c *Client) Authenticate() error {
	if c.conf.AccessToken != "" {
		c.logger.Println("✅ Access token found in config. Proceeding...")
		c.kc.SetAccessToken(c.conf.AccessToken)
		return nil
	}

	c.logger.Println("🔐 No access token found. Starting authentication flow...")

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

	c.conf.AccessToken = data.AccessToken
	if err := config.Save("config.yaml", c.conf); err != nil {
		return fmt.Errorf("failed to save access token to config: %v", err)
	}

	c.kc.SetAccessToken(c.conf.AccessToken)
	c.logger.Println("✅ Authentication successful! Access token saved to config.yaml")
	return nil
}

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
