package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/webauthn"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"

	"github.com/jgoulah/streamtime/internal/config"
)

// ServiceInfo holds information about a streaming service
type ServiceInfo struct {
	Name            string
	LoginURL        string
	TestURL         string
	CookieDomain    string
	RequiredCookies []string
	OnePasswordID   string // 1Password item ID for automated login
	AutoLogin       bool   // Whether to attempt automated login
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
		Name:            "Amazon Video",
		LoginURL:        "https://www.amazon.com/gp/video/settings/watch-history",
		TestURL:         "https://www.amazon.com/gp/video/settings/watch-history",
		CookieDomain:    ".amazon.com",
		RequiredCookies: []string{"session-id", "ubid-main"},
		OnePasswordID:   "kjsrkcoebve4nbdvdrszc36tc4",
		AutoLogin:       true,
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
	fmt.Println()
	fmt.Println("Press Enter to close the browser...")
	fmt.Scanln()
}

// get1PasswordCredentials retrieves username and password from 1Password
func get1PasswordCredentials(itemID string) (username, password string, err error) {
	// Get username/email
	cmd := exec.Command("op", "item", "get", itemID, "--fields", "label=email,label=username", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get credentials from 1Password: %w", err)
	}

	// Parse simple case - just get the value
	usernameStr := strings.TrimSpace(string(output))
	if strings.Contains(usernameStr, "jgoulah@gmail.com") {
		username = "jgoulah@gmail.com"
	}

	// Get password
	cmd = exec.Command("op", "item", "get", itemID, "--fields", "label=password")
	output, err = cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get password from 1Password: %w", err)
	}
	password = strings.TrimSpace(string(output))

	if username == "" || password == "" {
		return "", "", fmt.Errorf("failed to extract username or password from 1Password")
	}

	return username, password, nil
}

// get1PasswordOTP retrieves current OTP code from 1Password
func get1PasswordOTP(itemID string) (string, error) {
	cmd := exec.Command("op", "item", "get", itemID, "--otp")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get OTP from 1Password: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// automatedAmazonLogin performs automated login to Amazon using 1Password credentials
func automatedAmazonLogin(ctx context.Context, serviceInfo ServiceInfo) error {
	fmt.Println("🔐 Attempting automated login with 1Password credentials...")

	// Get credentials from 1Password
	username, password, err := get1PasswordCredentials(serviceInfo.OnePasswordID)
	if err != nil {
		return fmt.Errorf("failed to get 1Password credentials: %w", err)
	}
	fmt.Printf("✓ Retrieved credentials for %s\n", username)

	// Wait for Amazon login page and fill in email
	time.Sleep(2 * time.Second)
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`input[type="email"], input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[type="email"], input[name="email"]`, username, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to enter email: %w", err)
	}
	fmt.Println("✓ Entered email")
	
	// Click Continue/Submit button
	err = chromedp.Run(ctx,
		chromedp.Click(`input[id="continue"], input[type="submit"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to click continue: %w", err)
	}

	// Virtual authenticator suppresses passkey prompts, so just wait for password page
	fmt.Println("⏳ Waiting for password page...")
	time.Sleep(3 * time.Second)

	// Disable virtual authenticator now that we've bypassed the passkey prompt
	// This might help with password validation
	fmt.Println("🔧 Disabling virtual authenticator...")
	chromedp.Run(ctx, webauthn.Disable())

	var currentURL string
	chromedp.Run(ctx, chromedp.Location(&currentURL))
	fmt.Printf("📍 Current URL after email: %s\n", currentURL)

	if strings.Contains(currentURL, "/ap/signin") {
		fmt.Println("ℹ️  Passkey dismissed, re-entering credentials...")
		time.Sleep(2 * time.Second)

		// Re-enter email
		fmt.Println("📧 Looking for email field...")
		err = chromedp.Run(ctx,
			chromedp.WaitVisible(`input[type="email"], input[name="email"]`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("failed to find email field: %w", err)
		}
		fmt.Println("📧 Filling email field...")
		err = chromedp.Run(ctx,
			chromedp.SendKeys(`input[type="email"], input[name="email"]`, username, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("failed to type email: %w", err)
		}
		time.Sleep(1 * time.Second)
		fmt.Println("📧 Clicking continue...")
		err = chromedp.Run(ctx,
			chromedp.Click(`input[id="continue"], input[type="submit"]`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("failed to click continue: %w", err)
		}
		fmt.Println("✓ Re-entered email and clicked continue")
		time.Sleep(3 * time.Second)
	}

	// Wait for password field and enter password
	fmt.Println("🔑 Looking for password field...")
	time.Sleep(2 * time.Second)
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`input[type="password"], input[name="password"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to find password field: %w", err)
	}
	fmt.Println("🔑 Filling password field...")
	err = chromedp.Run(ctx,
		chromedp.SendKeys(`input[type="password"], input[name="password"]`, password, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to type password: %w", err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("✓ Entered password")
	
	// Click Sign In button
	fmt.Println("🔐 Clicking Sign In button...")
	err = chromedp.Run(ctx,
		chromedp.Click(`input[id="signInSubmit"], input[type="submit"], button[type="submit"]`, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("failed to click sign in: %w", err)
	}

	// Wait and check what page we land on
	fmt.Println("⏳ Waiting for sign in to process...")
	time.Sleep(5 * time.Second)

	currentURL = ""
	chromedp.Run(ctx, chromedp.Location(&currentURL))
	fmt.Printf("📍 After sign in click, current URL: %s\n", currentURL)

	// Check for error messages on page
	var errorText string
	chromedp.Run(ctx, chromedp.Evaluate(`
		(() => {
			const errBox = document.querySelector('#auth-error-message-box, .a-alert-error, [data-testid="error-message"]');
			return errBox ? errBox.innerText : '';
		})();
	`, &errorText))
	if errorText != "" {
		fmt.Printf("⚠️  Error message on page: %s\n", errorText)
	}

	// Check if OTP/2FA page appeared (multiple possible selectors)
	var otpVisible bool
	chromedp.Run(ctx,
		chromedp.Evaluate(`
			document.querySelector('input[name="otpCode"]') !== null ||
			document.querySelector('input[name="code"]') !== null ||
			document.querySelector('input[id="auth-mfa-otpcode"]') !== null
		`, &otpVisible),
	)

	// Also check URL for CVF (customer verification flow)
	if !otpVisible && strings.Contains(currentURL, "/ap/cvf") {
		fmt.Println("🔐 On verification page, checking for OTP field...")
		otpVisible = true
	}

	if otpVisible {
		fmt.Println("🔢 2FA required, retrieving OTP from 1Password...")
		otp, err := get1PasswordOTP(serviceInfo.OnePasswordID)
		if err != nil {
			return fmt.Errorf("failed to get OTP: %w", err)
		}
		fmt.Printf("✓ Retrieved OTP: %s\n", otp)

		// Try multiple OTP field selectors with longer timeout
		time.Sleep(2 * time.Second)
		fmt.Println("🔍 Looking for OTP field...")

		// Try each selector individually to see which one works
		var otpFieldFound bool
		selectors := []string{
			`input[name="otpCode"]`,
			`input[name="code"]`,
			`input[id="auth-mfa-otpcode"]`,
			`input[type="text"]`,  // CVF page might use generic text input
		}

		for _, selector := range selectors {
			err = chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery))
			if err == nil {
				fmt.Printf("✓ Found OTP field with selector: %s\n", selector)
				otpFieldFound = true

				// Fill the OTP
				err = chromedp.Run(ctx, chromedp.SendKeys(selector, otp, chromedp.ByQuery))
				if err != nil {
					return fmt.Errorf("failed to type OTP: %w", err)
				}
				break
			}
		}

		if !otpFieldFound {
			return fmt.Errorf("could not find OTP input field with any known selector")
		}

		time.Sleep(1 * time.Second)
		fmt.Println("🔍 Looking for submit button...")
		// Try to find and click submit button
		err = chromedp.Run(ctx,
			chromedp.Click(`input#auth-signin-button, button[type="submit"], input[type="submit"], button#a-autoid-0`, chromedp.ByQuery),
		)
		if err != nil {
			return fmt.Errorf("failed to click OTP submit: %w", err)
		}
		fmt.Println("✓ Entered OTP and clicked submit")
		time.Sleep(5 * time.Second)
	}

	// If on /ax/claim, wait for automatic redirect instead of navigating
	currentURL = ""
	chromedp.Run(ctx, chromedp.Location(&currentURL))

	if strings.Contains(currentURL, "/ax/claim") {
		fmt.Println("⏳ Waiting on authentication claim page for automatic redirect...")
		// Wait for redirect (Amazon will auto-redirect after processing auth)
		time.Sleep(10 * time.Second)

		// Check where we ended up
		currentURL = ""
		chromedp.Run(ctx, chromedp.Location(&currentURL))
		fmt.Printf("📍 After auth claim redirect: %s\n", currentURL)

		// If still on claim or back on signin, something went wrong
		if strings.Contains(currentURL, "/ax/claim") || strings.Contains(currentURL, "/ap/signin") {
			return fmt.Errorf("authentication did not complete - stuck on: %s", currentURL)
		}
	}

	// Only navigate if we're not already on watch history
	if !strings.Contains(currentURL, "/gp/video/settings/watch-history") {
		fmt.Println("🔄 Navigating to watch history...")
		time.Sleep(2 * time.Second)
		err = chromedp.Run(ctx,
			chromedp.Navigate(serviceInfo.TestURL),
			chromedp.WaitReady("body"),
		)
		if err != nil {
			return fmt.Errorf("failed to navigate to test URL: %w", err)
		}
	}
	
	// Wait for watch history page to fully load and set all cookies
	fmt.Println("⏳ Waiting for watch history page to fully load...")
	time.Sleep(10 * time.Second)

	// Debug: Check if we're actually logged in
	currentURL = ""
	chromedp.Run(ctx, chromedp.Location(&currentURL))
	fmt.Printf("📍 Current URL: %s\n", currentURL)

	if strings.Contains(currentURL, "/ap/signin") || strings.Contains(currentURL, "/ap/cvf") {
		return fmt.Errorf("still on login page - authentication may have failed")
	}

	fmt.Println("✓ Successfully logged in!")
	return nil
}

func exportCookies(serviceInfo ServiceInfo) ([]*network.Cookie, error) {
	// Create a context with non-headless browser for debugging
	// Enable testing/automation mode which uses virtual authenticators
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("enable-features", "WebAuthenticationTesting"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Add virtual authenticator via CDP to suppress WebAuthn prompts
	// This makes Chrome think it has a registered passkey device
	fmt.Println("🔧 Adding virtual authenticator to suppress passkey prompts...")
	err := chromedp.Run(ctx,
		webauthn.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Add a virtual authenticator with platform (internal) authenticator
			// This simulates having a built-in passkey like Touch ID
			opts := &webauthn.VirtualAuthenticatorOptions{
				Protocol:          webauthn.AuthenticatorProtocolCtap2,
				Transport:         webauthn.AuthenticatorTransportInternal,
				HasResidentKey:    true,
				HasUserVerification: true,
				IsUserVerified:    true,
				AutomaticPresenceSimulation: true,
			}

			_, err := webauthn.AddVirtualAuthenticator(opts).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to add virtual authenticator: %w", err)
			}
			return nil
		}),
	)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not set up virtual authenticator: %v\n", err)
		fmt.Println("   Continuing anyway - you may need to manually dismiss passkey prompts")
	} else {
		fmt.Println("✓ Virtual authenticator added successfully")
	}

	// Navigate to the login page
	fmt.Printf("🌐 Opening browser to %s\n", serviceInfo.LoginURL)

	err = chromedp.Run(ctx,
		chromedp.Navigate(serviceInfo.LoginURL),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	// Try automated login if supported
	if serviceInfo.AutoLogin && serviceInfo.OnePasswordID != "" {
		if err := automatedAmazonLogin(ctx, serviceInfo); err != nil {
			fmt.Printf("⚠️  Automated login failed: %v\n", err)
			fmt.Println("📝 Please log in manually...")
			fmt.Println()
			
			// Fall back to manual login
			fmt.Println("⏳ Waiting for you to log in...")
			if serviceInfo.TestURL != serviceInfo.LoginURL {
				fmt.Printf("   After logging in, navigate to: %s\n", serviceInfo.TestURL)
				fmt.Println("   Wait for the page to fully load, then press Enter here to continue...")
			} else {
				fmt.Println("   After logging in successfully, press Enter here to continue...")
			}
			fmt.Scanln()
		}
	} else {
		// Manual login for services without automated support
		fmt.Println("📝 Please log in to your account...")
		fmt.Println()
		fmt.Println("⏳ Waiting for you to log in...")
		if serviceInfo.TestURL != serviceInfo.LoginURL {
			fmt.Printf("   After logging in, navigate to: %s\n", serviceInfo.TestURL)
			fmt.Println("   Wait for the page to fully load, then press Enter here to continue...")
		} else {
			fmt.Println("   After logging in successfully, press Enter here to continue...")
		}
		fmt.Scanln()
	}

	fmt.Println("\n🔄 Extracting cookies...")
	
	// Debug: Compare cookies count before and after
	var cookiesBefore []*network.Cookie
	chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookiesBefore, _ = network.GetCookies().Do(ctx)
			return nil
		}),
	)
	fmt.Printf("📊 Cookie count after full page load: %d\n", len(cookiesBefore))
	
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
	// First show ALL cookies for debugging
	fmt.Println("\n🔍 ALL cookies from ALL domains:")
	for _, c := range cookies {
		fmt.Printf("  - %s (domain: %s)\n", c.Name, c.Domain)
	}
	fmt.Printf("\n🔍 Filtered cookies for %s:\n", serviceInfo.CookieDomain)
	for _, cookie := range cookies {
		if strings.Contains(cookie.Domain, strings.TrimPrefix(serviceInfo.CookieDomain, ".")) {
			fmt.Printf("  - %s (domain: %s, secure: %v, httpOnly: %v, expires: %v)\n", 
				cookie.Name, cookie.Domain, cookie.Secure, cookie.HTTPOnly, cookie.Expires)
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
