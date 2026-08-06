package scraper

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Message is a parsed channel post.
type Message struct {
	ID       int
	Text     string
	PhotoURL string
}

// ChannelScraper polls the public t.me embed preview page of a Telegram
// channel and emits new messages.
type ChannelScraper struct {
	channel  string
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger
	seen     map[int]struct{}
	baseline bool
}

// NewChannelScraper creates a scraper for the given public channel username.
func NewChannelScraper(channel string, interval, timeout time.Duration, logger *slog.Logger) *ChannelScraper {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChannelScraper{
		channel:  channel,
		interval: interval,
		logger:   logger,
		seen:     make(map[int]struct{}),
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
				DisableCompression:  true,
			},
		},
	}
}

// Start polls the channel and sends new messages to out until ctx is canceled.
// The first successful poll establishes a baseline (existing posts are not
// forwarded), subsequent polls emit only brand-new message IDs.
func (s *ChannelScraper) Start(ctx context.Context, out chan<- Message) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		s.poll(ctx, out)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *ChannelScraper) poll(ctx context.Context, out chan<- Message) {
	msgs, err := s.Fetch(ctx)
	if err != nil {
		s.logger.Warn("channel scrape failed", slog.String("err", err.Error()))
		return
	}

	if !s.baseline {
		s.baseline = true
		for _, m := range msgs {
			s.seen[m.ID] = struct{}{}
		}
		s.logger.Info("channel baseline established", slog.String("channel", s.channel), slog.Int("messages", len(msgs)))
		return
	}

	for _, m := range msgs {
		if _, ok := s.seen[m.ID]; ok {
			continue
		}
		s.seen[m.ID] = struct{}{}
		select {
		case out <- m:
		case <-ctx.Done():
			return
		}
	}
}

// Fetch retrieves and parses the current set of messages from the channel page.
func (s *ChannelScraper) Fetch(ctx context.Context) ([]Message, error) {
	url := "https://t.me/s/" + s.channel
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseMessages(body)
}

// parseMessages extracts channel posts from the t.me embed preview HTML.
func parseMessages(body []byte) ([]Message, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var msgs []Message
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if post := getAttr(n, "data-post"); post != "" {
				if id := postID(post); id != 0 {
					var msg Message
					msg.ID = id
					extractMessage(n, &msg)
					msgs = append(msgs, msg)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return msgs, nil
}

// extractMessage fills text and photo URL from a single message subtree.
func extractMessage(n *html.Node, msg *Message) {
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if class := getAttr(node, "class"); class != "" {
				if hasClass(class, "tgme_widget_message_text") {
					msg.Text = innerText(node)
				}
				if hasClass(class, "tgme_widget_message_photo_wrap") {
					if src := firstImgSrc(node); src != "" {
						msg.PhotoURL = src
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
}

func postID(post string) int {
	i := strings.LastIndexByte(post, '/')
	if i < 0 {
		return 0
	}
	id, err := strconv.Atoi(post[i+1:])
	if err != nil {
		return 0
	}
	return id
}

func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func hasClass(classList, target string) bool {
	for _, c := range strings.Fields(classList) {
		if c == target {
			return true
		}
	}
	return false
}

func firstImgSrc(n *html.Node) string {
	var found string
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if found != "" {
			return
		}
		if node.Type == html.ElementNode && node.Data == "img" {
			if src := getAttr(node, "src"); src != "" {
				found = src
				return
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return found
}

func innerText(n *html.Node) string {
	var sb strings.Builder
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
