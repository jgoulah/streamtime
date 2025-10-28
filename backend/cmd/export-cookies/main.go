package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/network"
)

func main() {
	// Create a context with a non-headless browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Navigate to myactivity.google.com
	log.Println("Opening browser...")
	log.Println("Please log in and navigate to: https://myactivity.google.com/product/youtube")
	log.Println("Once you're on the page, come back here and press Enter to export cookies...")

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://myactivity.google.com/product/youtube"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for user to log in
	fmt.Println("\nPress Enter when you're logged in and on the My Activity page...")
	fmt.Scanln()

	// Get all cookies
	var cookies []*network.Cookie
	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Filter for Google cookies only
	var googleCookies []*network.Cookie
	for _, cookie := range cookies {
		// Only include cookies from google.com domains
		if strings.Contains(cookie.Domain, "google.com") {
			googleCookies = append(googleCookies, cookie)
		}
	}

	log.Printf("\nFound %d Google cookies (including HTTPOnly)", len(googleCookies))
	fmt.Println("\n# Copy the output below into your config.yaml under youtube_tv.cookies:")
	fmt.Println("    cookies:")

	// Output in YAML format
	for _, cookie := range googleCookies {
		fmt.Printf("      - name: \"%s\"\n", cookie.Name)
		fmt.Printf("        value: \"%s\"\n", cookie.Value)
	}

	fmt.Println("\n✅ Cookie export complete!")
}
