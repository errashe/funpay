package main

import (
	"github.com/gocolly/colly"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EventType int

const (
	New EventType = iota
	Update
	Delete
	Initial
)

func (et EventType) String() string {
	switch et {
	case New: return "New"
	case Update: return "Update"
	case Delete: return "Delete"
	case Initial: return "Initial"
	default: return "Undefined"
	}
}

type Event struct {
	Type EventType
	Entry *Entry
}

type Parser struct {
	Collector *colly.Collector
	EventBus chan Event
	Entries Entries
	Tick time.Duration
	Initialized bool
}

func newParser() *Parser {
	parser := new(Parser)

	parser.Collector = colly.NewCollector(colly.AllowURLRevisit())
	parser.EventBus = make(chan Event)
	parser.Entries = make(map[string]*Entry)
	parser.Tick = 10*time.Second
	parser.Initialized = false

	parser.setup()

	return parser
}

func (p *Parser) setup() {
	p.Collector.OnHTML("a.tc-item", func(item *colly.HTMLElement) {
		sSide := item.ChildText("div.tc-side")
		sServer := item.ChildText("div.tc-server")
		sName := item.ChildText("div.tc-user .media-user-name span")
		sReviewsMedian := item.DOM.Find("div.tc-user .media-user-reviews .rating-stars .fas")
		sReviewsCount := item.ChildText("div.tc-user div.media-user-reviews span.rating-mini-count")
		sAmount := item.ChildText("div.tc-amount")
		sPrice := item.ChildText("div.tc-price")

		sAmount = strings.ReplaceAll(sAmount, " ", "")
		sPrice = strings.ReplaceAll(sPrice, " ", "")
		sPrice = strings.ReplaceAll(sPrice, string(rune(8381)), "")

		entry := new(Entry)
		entry.Side = sSide
		entry.Server = sServer
		entry.Seller = new(Seller)
		entry.Seller.Name = sName
		entry.Seller.ReviewsMedian = int64(sReviewsMedian.Length())
		entry.Seller.ReviewsCount, _ = strconv.ParseInt(sReviewsCount, 10, 64)
		entry.Amount, _ = strconv.ParseInt(sAmount, 10, 64)
		entry.Price, _ = strconv.ParseFloat(sPrice, 64)
		entry.Timestamp = time.Now()

		entry.ID = entry.getID()

		p.Proceed(entry)
	})

	p.Collector.OnScraped(func(response *colly.Response) {
		p.Clear()
		if !p.Initialized {
			p.Initialized = true
		}
	})
}

func (p *Parser) Proceed(entry *Entry) {
	currentEntry, exists := p.Entries[entry.getID()]
	if exists {
		currentEntry.Timestamp = entry.Timestamp
		if currentEntry.Amount != entry.Amount || currentEntry.Price != entry.Price {
			currentEntry.Amount = entry.Amount
			currentEntry.Price = entry.Price
			p.SendEvent(Update, currentEntry)
		}
	} else {
		p.Entries[entry.getID()] = entry
		p.SendEvent(New, entry)
	}
}

func (p *Parser) Clear() {
	for key, value := range p.Entries {
		if time.Since(value.Timestamp) > p.Tick {
			delete(p.Entries, key)
			p.SendEvent(Delete, value)
		}
	}
}

func (p *Parser) SendEvent(eventType EventType, entry *Entry) {
	if p.Initialized {
		p.EventBus <- Event{eventType, entry}
	}
}

func (p *Parser) Run(wg sync.WaitGroup) {
	defer wg.Done()

	p.Collector.Visit("https://funpay.ru/chips/2/")

	for range time.Tick(p.Tick) {
		p.Collector.Visit("https://funpay.ru/chips/2/")
	}
}
