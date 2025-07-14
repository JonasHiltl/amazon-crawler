package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/playwright-community/playwright-go"
)

type Product struct {
	ASIN                   string       `json:"asin"`
	Title                  string       `json:"title"`
	Description            string       `json:"description,omitempty"`
	AboutItem              string       `json:"aboutItem,omitempty"`
	Brand                  string       `json:"brand,omitempty"`
	Manufacturer           string       `json:"manufacturer,omitempty"`
	AgeRange               string       `json:"ageRange,omitempty"`
	Weight                 string       `json:"weight,omitempty"`
	Material               string       `json:"material,omitempty"`
	Color                  string       `json:"color,omitempty"`
	Origin                 string       `json:"origin,omitempty"`
	Dimensions             string       `json:"dimensions,omitempty"`
	SustainabilityFeatures []string     `json:"sustainabilityFeatures,omitempty"`
	AverageRating          float32      `json:"averageRating,omitempty"`
	Ratings                int          `json:"ratings,omitempty"`
	IsAmazonChoice         bool         `json:"isAmazonChoice"`
	Images                 []string     `json:"images,omitempty"`
	BoughtTogetherASINs    []string     `json:"boughtTogetherAsins,omitempty"`
	Categories             []string     `json:"categories,omitempty"`
	BestSellers            []BestSeller `json:"bestSellers,omitempty"`
	ListPrice              float32      `json:"listPrice,omitempty"`
	DiscountedPrice        float32      `json:"discountedPrice,omitempty"`
	Currency               string       `json:"currency,omitempty"`
	SellerID               string       `json:"sellerId,omitempty"`
	FirstAvailableAt       *time.Time   `json:"firstAvailableAt,omitempty"` // needs to be pointer, else won't be omitted if empty
	BoughtPastMonth        int          `json:"boughtPastMonth,omitempty"`
}

type BestSeller struct {
	Category string `json:"category"` // the category name, e.g. Baby, Baby Bottle Brushes
	Rank     int    `json:"rank"`
}

func ProductFromPage(page playwright.Page) (Product, error) {
	page.SetDefaultTimeout(2 * 1000) // 2 seconds

	asin, err := findASIN(page)
	if err != nil {
		return Product{}, err
	}

	log := slog.Default().With(slog.String("asin", asin))

	title, err := findTitle(page)
	if err != nil {
		return Product{}, err
	}
	description, err := findDescription(page)
	if err != nil {
		log.Debug(err.Error())
	}
	aboutItem, err := findAboutItem(page)
	if err != nil {
		log.Debug(err.Error())
	}
	brand, err := findBrand(page)
	if err != nil {
		log.Debug(err.Error())
	}
	manufacturer, err := findManufacturer(page)
	if err != nil {
		log.Debug(err.Error())
	}
	age, err := findAgeRange(page)
	if err != nil {
		log.Debug(err.Error())
	}
	color, err := findColor(page)
	if err != nil {
		log.Debug(err.Error())
	}
	material, err := findMaterial(page)
	if err != nil {
		log.Debug(err.Error())
	}
	weight, err := findWeight(page)
	if err != nil {
		log.Debug(err.Error())
	}
	dimensions, err := findDimensions(page)
	if err != nil {
		log.Debug(err.Error())
	}
	origin, err := findOrigin(page)
	if err != nil {
		log.Debug(err.Error())
	}
	rating, err := findAverageRating(page)
	if err != nil {
		log.Debug(err.Error())
	}
	ratingsAmount, err := findRatingsAmount(page)
	if err != nil {
		log.Debug(err.Error())
	}
	isAmazonChoice := findIsAmazonChoice(page)
	findSustainabilityFeatures, err := findSustainabilityFeatures(page)
	if err != nil {
		log.Debug(err.Error())
	}
	images, err := findImages(page)
	if err != nil {
		log.Debug(err.Error())
	}
	boughtTogether, err := findBoughtTogether(page)
	if err != nil {
		log.Debug(err.Error())
	}
	categories, err := findCategories(page)
	if err != nil {
		log.Debug(err.Error())
	}

	bestSellers := findBestsellers(page)
	price := findPrice(page)
	sellerID, err := findSellerID(page)
	if err != nil {
		slog.Debug(err.Error(), slog.String("asin", asin))
	}
	availableAt, err := findFirstAvailableAt(page)
	if err != nil {
		slog.Debug(err.Error(), slog.String("asin", asin))
	}
	boughtPastMonth, err := findBoughtPastMonth(page)
	if err != nil {
		slog.Debug(err.Error(), slog.String("asin", asin))
	}
	log.Debug("finished parsing all product fields")

	return Product{
		ASIN:                   asin,
		Title:                  title,
		Description:            description,
		AboutItem:              aboutItem,
		Brand:                  brand,
		Manufacturer:           manufacturer,
		AgeRange:               age,
		Weight:                 weight,
		Material:               material,
		Color:                  color,
		Origin:                 origin,
		Dimensions:             dimensions,
		AverageRating:          float32(rating),
		Ratings:                ratingsAmount,
		IsAmazonChoice:         isAmazonChoice,
		SustainabilityFeatures: findSustainabilityFeatures,
		Images:                 images,
		BoughtTogetherASINs:    boughtTogether,
		Categories:             categories,
		BestSellers:            bestSellers,
		ListPrice:              price.list,
		DiscountedPrice:        price.discounted,
		Currency:               price.currency,
		SellerID:               sellerID,
		FirstAvailableAt:       availableAt,
		BoughtPastMonth:        boughtPastMonth,
	}, nil
}

func findASIN(page playwright.Page) (string, error) {
	asin, err := AsinFromURL(page.URL())
	if err == nil {
		return asin, nil
	}

	asin, err = page.Locator("input#ASIN").GetAttribute("value")
	if err == nil && asin != "" {
		return asin, nil
	}

	asin, err = page.Locator("div#averageCustomerReviews").First().GetAttribute("data-asin")
	if err == nil && asin != "" {
		return asin, nil
	}

	return findProductStat(page, "ASIN")
}

func findTitle(page playwright.Page) (string, error) {
	text := getTextContent(page, "span#productTitle")
	if text != "" {
		return text, nil
	}

	return "", errors.New("title not found")
}

// test: B07VF1F52V
// test: B00M0DWQYI nested product description
// test: B07F8HTSKD, B0DSCDSZYG (aplus section)
func findDescription(page playwright.Page) (string, error) {
	// use inner text to filter out <script> like in B008CDR7LW
	desc := getInnerText(page, "div#productDescription")
	if desc != "" {
		return strings.TrimPrefix(desc, "Product Description"), nil
	}

	desc = getTextContent(page, "div#bookDescription_feature_div")
	if desc != "" {
		return desc, nil
	}

	// use innerText to filter out text inside <script> or <style> elements
	desc = getInnerText(page, "div#aplus:has-text(\"Product Description\")")
	if desc != "" {
		desc = strings.TrimPrefix(desc, "Product Description")
		return desc, nil
	}

	return "", errors.New("description not found")
}

func findAboutItem(page playwright.Page) (string, error) {
	text := getTextContent(page, "div#feature-bullets > ul", true)
	if text != "" {
		return text, nil
	}

	return "", errors.New("about item not found")
}

// test: B07F8HTSKD (from overview)
func findBrand(page playwright.Page) (string, error) {
	return findProductStat(page, "Brand")
}

// test: B0BKQDPP1Z (from information table)
// test: B07VF1F52V (from details bullet list)
// test: B0C7ZFCS2V (from information table)
func findManufacturer(page playwright.Page) (string, error) {
	return findProductStat(page, "Manufacturer")
}

// test: B0BKQDPP1Z (from glance_icons_div)
func findMaterial(page playwright.Page) (string, error) {
	return findProductStat(page, "Material", "Material Type", "Fabric type")
}

// test: B00I3K25R0 (Manufacturer recommended age)
// test: 0789436507 (Reading age)
// test: B08SGH7NKX (Age Range (Description))
func findAgeRange(page playwright.Page) (string, error) {
	return findProductStat(page, "Age Range", "Manufacturer recommended age", "Reading age")
}

// test: B08SGH7NKX (from overview)
// test: B089YNGH9K (from twister)
func findColor(page playwright.Page) (string, error) {
	color, err := findProductStat(page, "Color")
	if err == nil {
		return color, nil
	}

	container := page.Locator("div#inline-twister-dim-title-color_name")
	color = getTextContent(container, "span:last-child")
	if color != "" {
		return color, nil
	}
	return "", err
}

// test: B0DG2J2962 (from information table)
func findWeight(page playwright.Page) (string, error) {
	return findProductStat(page, "Item Weight", "Weight")
}

// test: B0DYJRDSRX (from overview)
func findDimensions(page playwright.Page) (string, error) {
	return findProductStat(page, "Product Dimensions", "Dimensions")
}

// test: B00I3K25R0 (from information table)
func findOrigin(page playwright.Page) (string, error) {
	return findProductStat(page, "Country/Region of origin", "Country of Origin")
}

func findAverageRating(page playwright.Page) (float64, error) {
	container := page.Locator("div#averageCustomerReviews")
	rating := getTextContent(container, "span:first-child a>span", true)
	if rating != "" {
		return strconv.ParseFloat(strings.TrimSpace(rating), 32)
	}
	return 0, errors.New("average rating not found")
}

func findRatingsAmount(page playwright.Page) (int, error) {
	rating := getTextContent(page, "span#acrCustomerReviewText", true)
	if rating != "" {
		parts := strings.Split(rating, " ")
		if len(parts) != 2 {
			return 0, fmt.Errorf("\"%s\" is an invalid rating text", rating)
		}
		amount := parts[0]
		return parseInt(amount)
	}
	return 0, errors.New("review amount not found")
}

// test: B00I3K25R0 (is amazon choice)
// test: B0B5S3HN9Q (no amazon choice)
func findIsAmazonChoice(page playwright.Page) bool {
	visible, err := page.Locator("div#acBadge_feature_div").IsVisible()
	return err == nil && visible
}

// test: B0CYC2N788 (Forestry practices)
// test: B0126LMDFK (4 features)
func findSustainabilityFeatures(page playwright.Page) ([]string, error) {
	container := page.Locator("div#climatePledgeFriendly").Locator("div.a-spacing-base").First()
	all, err := container.Locator("span.a-text-bold").All()
	if err != nil {
		return nil, errors.New("sustainability features not found")
	}
	features := make([]string, 0, len(all))
	for _, title := range all {
		text, err := title.TextContent()
		if err == nil && text != "" {
			features = append(features, strings.TrimSpace(text))
		}
	}
	return features, nil
}

func findImages(page playwright.Page) ([]string, error) {
	container := page.Locator("div#imageBlock")
	images, err := container.Locator("div#main-image-container>ul img").All()
	if err != nil {
		return nil, errors.New("images not found")
	}

	imgs := make([]string, 0, len(images))
	for _, img := range images {
		src, err := img.GetAttribute("src")
		if err == nil && src != "" {
			imgs = append(imgs, src)
		}
	}
	return imgs, nil
}

// test: B0126LMDFK (2 items)
func findBoughtTogether(page playwright.Page) ([]string, error) {
	container := page.Locator("div#similarities_feature_div").First()
	links, err := container.Locator("a").All()
	if err != nil {
		return nil, errors.New("bought together not found")
	}

	asins := mapset.NewThreadUnsafeSet[string]()
	for _, link := range links {
		href, err := link.GetAttribute("href")
		if err == nil && href != "" {
			asin, err := AsinFromURL(href)
			if err == nil {
				asins.Add(asin)
			}
		}
	}
	return asins.ToSlice(), nil
}
func findCategories(page playwright.Page) ([]string, error) {
	container := page.Locator("div#wayfinding-breadcrumbs_feature_div>ul")
	links, err := container.Locator("a").All()
	if err != nil {
		return nil, errors.New("categories not found")
	}

	categories := make([]string, 0, len(links))
	for _, link := range links {
		category, err := link.TextContent()
		if err == nil && category != "" {
			categories = append(categories, strings.TrimSpace(category))
		}
	}
	return categories, nil
}

// test: B07VF1F52V (from details bullet list)
// test: B0126LMDFK (from from information table)
func findBestsellers(page playwright.Page) []BestSeller {
	parseBestSeller := func(text string) (BestSeller, error) {
		// Extract rank using regex
		re := regexp.MustCompile(`#([\d,]+)`)
		match := re.FindStringSubmatch(text)
		if len(match) < 2 {
			return BestSeller{}, fmt.Errorf("could not find rank in string")
		}
		rank, err := parseInt(match[1])
		if err != nil {
			return BestSeller{}, err
		}

		// extract category name from 2 variants:
		// #199 in Office Products (See Top 100 in Office Products)
		// #3 in Office Laminating Supplies
		start := strings.Index(text, "in")
		end := strings.Index(text, "(")

		start += len("in")
		var categoryName string
		if end != -1 { // "(...)" part is not always there, so end might be missing
			categoryName = strings.TrimSpace(text[start:end])
		} else {
			categoryName = strings.TrimSpace(text[start:])
		}
		return BestSeller{
			Rank:     rank,
			Category: categoryName,
		}, nil
	}

	var bestSellers []BestSeller
	const bestSellerLabel = "Best Sellers Rank"
	var rawRanks string

	// first try to set whole text from the information table
	container := page.Locator("div:is(#prodDetails, #technicalSpecifications_feature_div)")
	selector := fmt.Sprintf("tr:has-text(\"%s\")", bestSellerLabel)
	row := container.Locator(selector)
	rawRanks = getTextContent(row, "td")

	// then try the details bullet list
	if rawRanks == "" {
		container = page.Locator("div#detailBulletsWrapper_feature_div")
		selector = fmt.Sprintf("ul > li:has-text(\"%s\")", bestSellerLabel)
		// the whole list item content, including multiple best seller ranks and the label
		text := getTextContent(container, selector)
		parts := strings.Split(text, ":")
		if len(parts) > 1 {
			rawRanks = strings.Join(parts[1:], ":") // join parts so value can include ":" char
		}
	}

	// rawRanks includes multiple best seller ranks
	// need to split by "#" to get each category
	for bestSellerText := range strings.SplitSeq(rawRanks, "#") {
		bestSeller, err := parseBestSeller("#" + bestSellerText) // prepend "#" back to be able to parse rank
		if err == nil {
			bestSellers = append(bestSellers, bestSeller)
		}
	}

	return bestSellers
}

type price struct {
	list       float32
	discounted float32
	currency   string
}

// test: B0DG2J2962 (no discount)
func findPrice(page playwright.Page) price {
	container := page.Locator("div:is(#corePriceDisplay_desktop_feature_div, #corePrice_desktop)").First()
	currency := getTextContent(container, ".a-price-symbol", true)
	price := price{
		currency: currency,
	}

	dicountedContainer := container.Locator(".priceToPay").First()
	whole := getTextContent(dicountedContainer, ".a-price-whole", true)
	fraction := getTextContent(dicountedContainer, ".a-price-fraction", true)
	if whole != "" {
		// whole should already contain an ending "." character, else append it
		if whole[len(whole)-1] != '.' {
			whole += "."
		}
		discountedPrice := strings.TrimSpace(whole + fraction)
		discountedPriceF, err := strconv.ParseFloat(discountedPrice, 32)
		if err == nil {
			price.discounted = float32(discountedPriceF)
		}
	}

	listPriceContainer := container.Locator("div:last-child")
	listPrice := getTextContent(listPriceContainer, ".a-text-price .a-offscreen", true)
	listPrice = strings.TrimPrefix(listPrice, currency)
	listPriceF, err := strconv.ParseFloat(listPrice, 32)
	if err == nil {
		price.list = float32(listPriceF)
	}

	return price
}

// test: B0074TRKFI (sellerID is ATVPDKIKX0DER)
// test: B0DPLTD14T (sellerID is A34ATOKEXB1ZYM)
func findSellerID(page playwright.Page) (string, error) {
	// try to get from js var
	result, err := page.Evaluate(`() => ue_mid`)
	if err == nil {
		mID, ok := result.(string)
		if ok && mID != "" {
			return mID, nil
		}
	}

	// infinitely hangs
	/*
		// try the hidden input field
		sellerID, err := page.Locator("input#merchantID").GetAttribute("value")
		if err == nil && sellerID != "" {
			return sellerID, nil
		}
		slog.Debug("not input with merchantID found")
	*/

	link := page.Locator("a#sellerProfileTriggerId")
	visible, err := link.IsVisible()
	if visible && err == nil {
		href, err := link.First().GetAttribute("href")
		if err == nil && href != "" {
			base := "https://www.amazon.com"
			fullURL := base + href
			parsedURL, err := url.Parse(fullURL)
			if err == nil {
				sellerID := parsedURL.Query().Get("seller")
				if sellerID != "" {
					return sellerID, nil
				}
			}
		}
	}

	return "", errors.New("seller id not found")
}

// test: B0BGYK6SVQ (Date First Available)
// test: 0679805273 (Publication date)
func findFirstAvailableAt(page playwright.Page) (*time.Time, error) {
	date, err := findProductStat(page, "Date First Available", "Publication date", "Release date")
	if err != nil {
		return nil, err
	}
	const layout = "January 2, 2006"
	parsed, err := time.Parse(layout, date)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// test: B0DG2J2962 (1k)
func findBoughtPastMonth(page playwright.Page) (int, error) {
	socialProof := getTextContent(page, "span#social-proofing-faceout-title-tk_bought")
	if socialProof != "" {
		// split the first word from the remaining text
		parts := strings.SplitN(socialProof, " ", 2)
		// the amount is the first word in the string
		amount := strings.Replace(parts[0], "+", "", 1)
		return parseInt(amount)
	}
	return 0, errors.New("bought past month not found")
}

// Searches in multiple locations for the product information by the name of the info,
// e.g. Manufacturer, Country of Origin, Brand
func findProductStat(page playwright.Page, names ...string) (string, error) {
	for _, name := range names {
		stat := findProductStatOverview_Page(page, name)
		if stat != "" {
			return stat, nil
		}

		stat = findProductStatGlanceIcons_Page(page, name)
		if stat != "" {
			return stat, nil
		}

		stat = findProductStatBulletList_Page(page, name)
		if stat != "" {
			return stat, nil
		}

		stat = findProductStatInformationTable_Page(page, name)
		if stat != "" {
			return stat, nil
		}
	}

	return "", fmt.Errorf("%s not found", names[0])
}

// test: B0BKQDPP1Z has material in glance_icons_div
func findProductStatOverview_Page(page playwright.Page, name string) string {
	container := page.Locator("div#productOverview_feature_div")
	selector := fmt.Sprintf("tr:has-text(\"%s\")", name)
	row := container.Locator(selector).First()
	return getTextContent(row, "td:nth-child(2)")
}

// Finds Products stats from the glances_icons section.
// See product B0BKQDPP1Z to view a glances_icons section
func findProductStatGlanceIcons_Page(page playwright.Page, name string) string {
	// the section has nested tables
	container := page.Locator("div#glance_icons_div")
	selector := fmt.Sprintf("table table tr:has-text(\"%s\")", name)
	row := container.Locator(selector).First()
	return getTextContent(row, "td:nth-child(2) > span:last-child")
}

// Searches for product information in the bullet list available for some products.
// E.g. found on B07VF1F52V
func findProductStatBulletList_Page(page playwright.Page, name string) string {
	selector := fmt.Sprintf("div#detailBulletsWrapper_feature_div ul > li > span:has-text(\"%s\")", name)
	item := page.Locator(selector)
	return getTextContent(item, "span:last-child")
}

// Searches for product information in the "Product information" table available for most products.
// E.g. found on B0BKQDPP1Z
func findProductStatInformationTable_Page(page playwright.Page, name string) string {
	container := page.Locator("div:is(#prodDetails, #technicalSpecifications_feature_div)")
	selector := fmt.Sprintf("tr:has-text(\"%s\")", name)
	rows, err := container.Locator(selector).All()
	if err != nil {
		return ""
	}

	// we need to ensure the head fully matches name to filter our false positives.
	// E.g. "Manufacturer" might find "Manufacturer recommended age" first
	for _, row := range rows {
		head := getTextContent(row, "th")
		if strings.EqualFold(name, head) {
			return getTextContent(row, "td:last-child", true)
		}
	}
	return ""
}

func getTextContent(target any, selector string, first ...bool) string {
	useFirst := len(first) > 0 && first[0]

	var elem playwright.Locator
	switch t := target.(type) {
	case playwright.Page:
		if useFirst {
			elem = t.Locator(selector).First()
		} else {
			elem = t.Locator(selector)
		}
	case playwright.Locator:
		if useFirst {
			elem = t.Locator(selector).First()
		} else {
			elem = t.Locator(selector)
		}
	default:
		return ""
	}

	visible, _ := elem.IsVisible() // first check visibility, else playwright waits for element to appear till timeout is hit
	if visible {
		text, _ := elem.TextContent()
		return strings.TrimSpace(text)
	}

	return ""
}

func getInnerText(target any, selector string, first ...bool) string {
	useFirst := len(first) > 0 && first[0]

	var elem playwright.Locator
	switch t := target.(type) {
	case playwright.Page:
		if useFirst {
			elem = t.Locator(selector).First()
		} else {
			elem = t.Locator(selector)
		}
	case playwright.Locator:
		if useFirst {
			elem = t.Locator(selector).First()
		} else {
			elem = t.Locator(selector)
		}
	default:
		return ""
	}

	visible, _ := elem.IsVisible()
	if visible {
		text, _ := elem.InnerText()
		return strings.TrimSpace(text)
	}
	return ""
}

// Parses a lot of number formats to a go int.
// Supports the units 1K, 1M.
// Supporst . or ,
func parseInt(text string) (int, error) {
	s := strings.ToLower(strings.TrimSpace(text))
	var hasUnit bool
	var multiplier float64 = 1

	switch {
	case strings.HasSuffix(s, "k"):
		multiplier = 1000
		s = strings.TrimSuffix(s, "k")
		hasUnit = true
	case strings.HasSuffix(s, "m"):
		multiplier = 1000000
		s = strings.TrimSuffix(s, "m")
		hasUnit = true
	}

	if hasUnit {
		// treat "," as decimal separator
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		// treat "," as thousands separators
		s = strings.ReplaceAll(s, ",", "")
		// treat "." as thousands separators
		s = strings.ReplaceAll(s, ".", "")
	}

	// Try parsing as float (in case of decimal k/m values)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.New("invalid number format")
	}

	return int(f * multiplier), nil
}

func AsinFromURL(url string) (string, error) {
	re := regexp.MustCompile(`dp(?:\/|%2[Ff])([A-Z0-9]{10})`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}
	return "", errors.New("url doesn't contain asin")
}
