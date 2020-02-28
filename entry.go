package main

import (
	"crypto/md5"
	"fmt"
	"time"
)

type Seller struct {
	Name string
	ReviewsCount int64
	ReviewsMedian int64
}

type Entry struct {
	ID string
	Server string
	Side string
	Seller *Seller
	Amount int64
	Price float64
	Timestamp time.Time
}

func (e *Entry) getID() string {
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%s", e.Side, e.Server, e.Seller.Name)))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

type Entries map[string]*Entry