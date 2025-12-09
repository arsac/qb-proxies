package proxy

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/arsac/qb-proxies/internal/config"
	"github.com/arsac/qb-proxies/internal/transform"
)

type Handler struct {
	feeds  map[string]*feedProxy
	client *http.Client
}

type feedProxy struct {
	config config.Feed
	engine *transform.Engine
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	h := &Handler{
		feeds: make(map[string]*feedProxy),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, feed := range cfg.Feeds {
		engine, err := transform.NewEngine()
		if err != nil {
			return nil, fmt.Errorf("creating transform engine for %s: %w", feed.Name, err)
		}

		for _, t := range feed.Transformations {
			key := fmt.Sprintf("%s_%s", feed.Name, t.Field)
			if err := engine.Compile(key, t.Expression); err != nil {
				return nil, fmt.Errorf("compiling transformation for %s.%s: %w", feed.Name, t.Field, err)
			}
		}

		h.feeds[feed.Path] = &feedProxy{
			config: feed,
			engine: engine,
		}
	}

	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/healthz", "/readyz", "/livez":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	fp, ok := h.feeds[path]
	if !ok {
		http.NotFound(w, r)
		return
	}

	log.Printf("proxying %s -> %s", path, fp.config.Upstream)

	resp, err := h.client.Get(fp.config.Upstream)
	if err != nil {
		log.Printf("error fetching upstream: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading upstream: %v", err)
		http.Error(w, "read error", http.StatusBadGateway)
		return
	}

	transformed, err := h.transformFeed(fp, body)
	if err != nil {
		log.Printf("error transforming feed: %v", err)
		http.Error(w, "transform error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(transformed)
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
	Attrs   []xml.Attr `xml:",any,attr"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	GUID        string     `xml:"guid"`
	PubDate     string     `xml:"pubDate"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	// Custom fields for non-standard feeds (e.g., Academic Torrents)
	InfoHash string `xml:"infohash,omitempty"`
	Size     string `xml:"size,omitempty"`
	Category string `xml:"category,omitempty"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func (h *Handler) transformFeed(fp *feedProxy, data []byte) ([]byte, error) {
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("parsing rss: %w", err)
	}

	for i := range rss.Channel.Items {
		item := &rss.Channel.Items[i]
		if err := h.transformItem(fp, item); err != nil {
			return nil, fmt.Errorf("transforming item %d: %w", i, err)
		}
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling rss: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}

func (h *Handler) transformItem(fp *feedProxy, item *Item) error {
	rssItem := &transform.RSSItem{
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		GUID:        item.GUID,
		PubDate:     item.PubDate,
		InfoHash:    item.InfoHash,
		Size:        item.Size,
		Category:    item.Category,
	}
	if item.Enclosure != nil {
		rssItem.Enclosure = transform.Enclosure{
			URL:    item.Enclosure.URL,
			Length: item.Enclosure.Length,
			Type:   item.Enclosure.Type,
		}
	}

	for _, t := range fp.config.Transformations {
		key := fmt.Sprintf("%s_%s", fp.config.Name, t.Field)
		result, err := fp.engine.Eval(key, rssItem)
		if err != nil {
			return fmt.Errorf("evaluating %s: %w", t.Field, err)
		}

		switch strings.ToLower(t.Field) {
		case "title":
			item.Title = result
			rssItem.Title = result
		case "link":
			item.Link = result
			rssItem.Link = result
		case "description":
			item.Description = result
			rssItem.Description = result
		case "guid":
			item.GUID = result
			rssItem.GUID = result
		case "pubdate":
			item.PubDate = result
			rssItem.PubDate = result
		case "enclosureurl", "enclosure.url":
			if item.Enclosure == nil {
				item.Enclosure = &Enclosure{}
			}
			item.Enclosure.URL = result
			rssItem.Enclosure.URL = result
		case "enclosurelength", "enclosure.length":
			if item.Enclosure == nil {
				item.Enclosure = &Enclosure{}
			}
			item.Enclosure.Length = result
			rssItem.Enclosure.Length = result
		case "enclosuretype", "enclosure.type":
			if item.Enclosure == nil {
				item.Enclosure = &Enclosure{}
			}
			item.Enclosure.Type = result
			rssItem.Enclosure.Type = result
		}
	}

	return nil
}
