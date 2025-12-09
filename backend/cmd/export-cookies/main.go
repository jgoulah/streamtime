package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"

	"github.com/jgoulah/streamtime/internal/config"
)

// ServiceInfo holds information about a streaming service
type ServiceInfo struct {
	Name        string
	LoginURL    string
	TestURL     string
	CookieDomain string
	RequiredCookies []string
}

var services = map[string]ServiceInfo{
	"netflix": {
		Name:        "Netflix",
		LoginURL:    "https://www.netflix.com/login",
		TestURL:     "https://www.netflix.com/viewingactivity",
		CookieDomain: ".netflix.com",
		RequiredCookies: []string{"NetflixId", "SecureNetflixId"},
	},
	"youtube_tv": {
		Name:        "YouTube TV",
		LoginURL:    "https://myactivity.google.com/product/youtube",
		TestURL:     "https://myactivity.google.com/product/youtube",
		CookieDomain: ".google.com",
		RequiredCookies: []string{"SID", "HSID", "SSID", "APISID", "SAPISID"},
	},
	"amazon_video": {
		Name:        "Amazon Video",
		LoginURL:    "https://www.amazon.com/gp/video/settings/watch-history",
		TestURL:     "https://www.amazon.com/gp/video/settings/watch-history",
		CookieDomain: ".amazon.com",
		RequiredCookies: []string{"session-id", "ubid-main"},
	},
	"hulu": {
		Name:        "Hulu",
		LoginURL:    "https://auth.hulu.com/web/login/enter-email",
		TestURL:     "https://www.hulu.com/hub/home/collection/282",
		CookieDomain: ".hulu.com",
		RequiredCookies: []string{"hulu_session"},
	},
}

func main() {
	// Parse command-line flags
	serviceFlag := flag.String("service", "", "Service to export cookies for (netflix, youtube_tv, amazon_video, hulu)")
	configPath := flag.String("config", "./config.yaml", "Path to config.yaml file")
	validateOnly := flag.Bool("validate", false, "Only validate existing cookies without exporting new ones")
	flag.Parse()

	if *serviceFlag == "" {
		fmt.Println("❌ Error: --service flag is required")
		fmt.Println("\nUsage:")
		fmt.Println("  ./export-cookies --service netflix")
		fmt.Println("  ./export-cookies --service youtube_tv")
		fmt.Println("  ./export-cookies --service amazon_video")
		fmt.Println("  ./export-cookies --service hulu")
		fmt.Println("  ./export-cookies --service netflix --validate")
		os.Exit(1)
	}

	serviceKey := strings.ToLower(*serviceFlag)
	serviceInfo, ok := services[serviceKey]
	if !ok {
		fmt.Printf("❌ Error: Unknown service '%s'\n", serviceKey)
		fmt.Println("\nSupported services: netflix, youtube_tv, amazon_video, hulu")
		os.Exit(1)
	}

	// Validate only mode
	if *validateOnly {
		fmt.Printf("🔍 Validating %s cookies...\n", serviceInfo.Name)
		if err := validateCookies(*configPath, serviceKey, serviceInfo); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Cookies are valid!")
		os.Exit(0)
	}

	// Export mode
	fmt.Printf("🔧 Export Cookies Tool for %s\n", serviceInfo.Name)
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	cookies, err := exportCookies(serviceInfo)
	if err != nil {
		log.Fatalf("❌ Failed to export cookies: %v", err)
	}

	fmt.Printf("\n✅ Successfully exported %d cookies\n", len(cookies))

	// Validate the cookies
	fmt.Println("\n🔍 Validating cookies...")
	if err := testCookies(cookies, serviceInfo); err != nil {
		fmt.Printf("⚠️  Warning: Cookie validation failed: %v\n", err)
		fmt.Println("The cookies were exported but may not work correctly.")
		fmt.Println("You may need to try logging in again.")
		fmt.Println()
	} else {
		fmt.Println("✅ Cookies validated successfully!")
		fmt.Println()
	}

	// Update config file
	fmt.Printf("💾 Updating %s...\n", *configPath)
	if err := updateConfig(*configPath, serviceKey, cookies); err != nil {
		log.Fatalf("❌ Failed to update config: %v", err)
	}

	fmt.Println("✅ Configuration updated successfully!")
	fmt.Println()
	fmt.Printf("🎉 Done! You can now run the %s scraper.\n", serviceInfo.Name)
}

func exportCookies(serviceInfo ServiceInfo) ([]*network.Cookie, error) {
	// Create a context with a non-headless browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Navigate to the login page
	fmt.Printf("🌐 Opening browser to %s\n", serviceInfo.LoginURL)
	fmt.Println("📝 Please log in to your account...")
	fmt.Println()

	err := chromedp.Run(ctx,
		chromedp.Navigate(serviceInfo.LoginURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for user to log in
	fmt.Println("⏳ Waiting for you to log in...")
	if serviceInfo.TestURL != serviceInfo.LoginURL {
		fmt.Printf("   After logging in, navigate to: %s\n", serviceInfo.TestURL)
		fmt.Println("   Wait for the page to fully load, then press Enter here to continue...")
	} else {
		fmt.Println("   After logging in successfully, press Enter here to continue...")
	}
	fmt.Scanln()

	fmt.Println("\n🔄 Extracting cookies...")

	// Get all cookies
	var cookies []*network.Cookie
	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	// Filter for service-specific cookies
	var filteredCookies []*network.Cookie
	for _, cookie := range cookies {
		if strings.Contains(cookie.Domain, strings.TrimPrefix(serviceInfo.CookieDomain, ".")) {
			filteredCookies = append(filteredCookies, cookie)
		}
	}

	if len(filteredCookies) == 0 {
		return nil, fmt.Errorf("no cookies found for domain %s - make sure you're logged in", serviceInfo.CookieDomain)
	}

	// Check for required cookies
	cookieNames := make(map[string]bool)
	for _, cookie := range filteredCookies {
		cookieNames[cookie.Name] = true
	}

	var missingRequired []string
	for _, required := range serviceInfo.RequiredCookies {
		if !cookieNames[required] {
			missingRequired = append(missingRequired, required)
		}
	}

	if len(missingRequired) > 0 {
		fmt.Printf("⚠️  Warning: Missing some expected cookies: %v\n", missingRequired)
		fmt.Println("   The scraper may not work correctly. Try logging in again.")
	}

	return filteredCookies, nil
}

func testCookies(cookies []*network.Cookie, serviceInfo ServiceInfo) error {
	// Create a new headless browser to test the cookies
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Set a timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// First navigate to the domain to set cookies
	baseDomain := "https://" + strings.TrimPrefix(serviceInfo.CookieDomain, ".")
	err := chromedp.Run(ctx, chromedp.Navigate(baseDomain))
	if err != nil {
		return fmt.Errorf("failed to navigate to base domain: %w", err)
	}

	// Set the cookies
	for _, cookie := range cookies {
		expr := network.SetCookie(cookie.Name, cookie.Value).
			WithDomain(cookie.Domain).
			WithPath(cookie.Path).
			WithSecure(cookie.Secure).
			WithHTTPOnly(cookie.HTTPOnly)

		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			return expr.Do(ctx)
		})); err != nil {
			return fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err)
		}
	}

	// Navigate to the test page
	var pageTitle string
	err = chromedp.Run(ctx,
		chromedp.Navigate(serviceInfo.TestURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Title(&pageTitle),
	)
	if err != nil {
		return fmt.Errorf("failed to test cookies: %w", err)
	}

	// Check if we're on a login page (basic check)
	pageTitle = strings.ToLower(pageTitle)
	if strings.Contains(pageTitle, "sign in") || strings.Contains(pageTitle, "log in") || strings.Contains(pageTitle, "login") {
		return fmt.Errorf("cookies appear invalid - redirected to login page (title: %s)", pageTitle)
	}

	return nil
}

func validateCookies(configPath, serviceKey string, serviceInfo ServiceInfo) error {
	// Load current config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	serviceCfg, ok := cfg.Services[serviceKey]
	if !ok {
		return fmt.Errorf("service %s not found in config", serviceKey)
	}

	if len(serviceCfg.Cookies) == 0 {
		return fmt.Errorf("no cookies configured for %s", serviceInfo.Name)
	}

	// Convert to network.Cookie format
	var cookies []*network.Cookie
	for _, c := range serviceCfg.Cookies {
		cookies = append(cookies, &network.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: serviceInfo.CookieDomain,
			Path:   "/",
			Secure: true,
		})
	}

	return testCookies(cookies, serviceInfo)
}

func updateConfig(configPath, serviceKey string, cookies []*network.Cookie) error {
	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Get services section
	servicesMap, ok := configMap["services"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("services section not found in config")
	}

	// Get or create the service section
	serviceMap, ok := servicesMap[serviceKey].(map[string]interface{})
	if !ok {
		serviceMap = make(map[string]interface{})
		servicesMap[serviceKey] = serviceMap
	}

	// Convert cookies to config format
	var cookieList []map[string]string
	for _, cookie := range cookies {
		cookieList = append(cookieList, map[string]string{
			"name":  cookie.Name,
			"value": cookie.Value,
		})
	}

	// Update cookies in the service
	serviceMap["cookies"] = cookieList

	// Ensure enabled is set to true
	serviceMap["enabled"] = true

	// Write back to file
	output, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
