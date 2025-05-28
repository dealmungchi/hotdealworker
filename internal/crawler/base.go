package crawler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/logger"

	"github.com/PuerkitoBio/goquery"
)

// ============================================================================
// DOCUMENT AND DEAL PROCESSING
// ============================================================================

// createDocument creates a goquery document from a reader
func (c *BaseCrawler) createDocument(reader io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 오류: %v", err)
	}
	return doc, nil
}

// processDeals processes deals in parallel using goroutines
func (c *BaseCrawler) processDeals(selections *goquery.Selection, processor ProcessorFunc) []HotDeal {
	dealChan := make(chan *HotDeal, selections.Length())
	var wg sync.WaitGroup

	selections.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(s *goquery.Selection) {
			defer wg.Done()

			deal, err := processor(s)
			if err != nil {
				logger.Error("[%s] Error processing deal: %v", c.Provider, err)
				return
			}

			if deal != nil {
				dealChan <- deal
			}
		}(s)
	})

	wg.Wait()
	close(dealChan)

	// Collect the processed deals
	var deals []HotDeal
	for deal := range dealChan {
		if deal != nil {
			deals = append(deals, *deal)
		}
	}

	return deals
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// GetName returns the crawler's type name for logging
func (c *BaseCrawler) GetName() string {
	if c.Provider != "" {
		return c.Provider + "Crawler"
	}
	return reflect.TypeOf(c).Elem().Name()
}

// GetProvider returns the provider name for the crawler
func (c *BaseCrawler) GetProvider() string {
	return c.Provider
}

// ResolveURL resolves a relative URL against the base URL
func (c *BaseCrawler) ResolveURL(href string) string {
	if href == "" {
		return ""
	}

	// 이미 스킴이 있는 절대 URL
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// 프로토콜 상대 URL
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	// 스킴 없는 절대 URL (도메인처럼 보이는 경우만)
	if isLikelyDomainURL(href) {
		return "https://" + href
	}

	// 상대 경로 처리
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = c.URL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		logger.Error("Error parsing base URL '%s': %v", baseURL, err)
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		logger.Error("Error parsing href '%s': %v", href, err)
		return href
	}

	return base.ResolveReference(ref).String()
}

// ExtractPrice extracts the price from a title using the configured regex
func (c *BaseCrawler) ExtractPrice(title string) (string, string) {
	if c.PriceRegex == "" || title == "" {
		return title, ""
	}

	re := regexp.MustCompile(c.PriceRegex)
	if match := re.FindStringSubmatch(title); len(match) > 1 {
		price := strings.TrimSpace(match[1])
		return title, price
	}

	return title, ""
}

// ProcessImage fetches an image and converts it to base64
func (c *BaseCrawler) ProcessImage(imageURL string) (string, string, error) {
	imageURL = c.ResolveURL(imageURL)
	if imageURL == "" {
		return "", "", nil
	}

	var data []byte
	var err error
	if c.Provider == "Bbasak" {
		data, err = helpers.FetchSimply(imageURL, http.Header{
			"Referer": []string{"https://bbasak.com/bbs/board.php?bo_table=bbasak1"},
		})
	} else {
		data, err = helpers.FetchSimply(imageURL)
	}

	if err != nil {
		logger.Warn("Error fetching image: %v", err)
		return "", "", nil
	}

	return base64.StdEncoding.EncodeToString(data), imageURL, nil
}

// CreateDeal creates a HotDeal with the given properties
func (c *BaseCrawler) CreateDeal(id, title, link, price, thumbnail, thumbnailLink, postedAt, category string) *HotDeal {
	return &HotDeal{
		Id:            id,
		Title:         title,
		Link:          link,
		Price:         price,
		Thumbnail:     thumbnail,
		ThumbnailLink: thumbnailLink,
		PostedAt:      postedAt,
		Category:      category,
		Provider:      c.Provider,
	}
}

// ExtractURLFromStyle extracts a URL from a CSS style attribute
func (c *BaseCrawler) ExtractURLFromStyle(style string) string {
	re := regexp.MustCompile(`url\((?:['"]?)(.*?)(?:['"]?)\)`)
	if matches := re.FindStringSubmatch(style); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Debug logs a debug message
func (b *BaseCrawler) Debug(format string, v ...interface{}) {
	logger.Debug("[%s] "+format, append([]interface{}{b.GetName()}, v...)...)
}

// isLikelyDomainURL checks if a string looks like a domain URL
func isLikelyDomainURL(href string) bool {
	domainLike := regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/|$)`)
	return domainLike.MatchString(href)
}

// ============================================================================
// CATEGORY CLASSIFICATION
// ============================================================================

// classifyCategory classifies deal categories into standardized groups
func classifyCategory(category string) string {
	if category == "" {
		return "기타"
	}

	mapping := map[string][]string{
		"전자제품/디지털/PC/하드웨어": {
			"PC/하드웨어", "PC관련", "컴퓨터", "디지털", "PC제품", "전자제품", "가전제품", "가전",
			"모바일", "노트북/모바일", "휴대폰", "A/V", "VR", "게임H/W", "PC 하드웨어", "모바일 / 가젯",
		},
		"소프트웨어/게임": {
			"SW/게임", "게임", "게임/SW", "게임S/W", "게임 / SW",
		},
		"생활용품/인테리어/주방": {
			"생활용품", "인테리어", "주방용품", "생활/식품", "가구인테리어",
		},
		"식품/먹거리": {
			"식품", "음식", "먹거리", "식품/건강", "식품/식당",
		},
		"의류/패션/잡화": {
			"의류", "의류/잡화", "패션/의류", "패션소품", "잡화", "신발", "가방/지갑",
			"명품", "시계/쥬얼리", "패션잡화",
		},
		"화장품/뷰티": {
			"화장품", "뷰티/미용", "화장품/바디",
		},
		"도서/미디어/콘텐츠": {
			"도서", "서적", "도서/미디어",
		},
		"카메라/사진": {
			"카메라", "카메라/사진",
		},
		"상품권/쿠폰/포인트": {
			"상품권/쿠폰", "모바일/상품권", "쿠폰", "포인트/래플", "패키지/이용권",
		},
		"출산/육아": {
			"육아", "출산육아", "육아용품",
		},
		"반려동물": {
			"애완용품", "반려동물용품",
		},
		"스포츠/아웃도어/레저": {
			"등산/캠핑", "스포츠용품", "레저용품",
		},
		"건강/비타민": {
			"비타민/의약", "영양제",
		},
		"여행/서비스": {
			"여행", "여행/서비스",
		},
		"이벤트/응모/바이럴": {
			"이벤트", "응모", "바이럴",
		},
		"학용품/사무용품": {
			"학용/사무용품",
		},
	}

	// 매핑 확인
	for bigCategory, keywords := range mapping {
		for _, keyword := range keywords {
			if strings.Contains(category, keyword) {
				return bigCategory
			}
		}
	}

	return "기타"
}
