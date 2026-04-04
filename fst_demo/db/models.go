package db

import (
	"strconv"
	"strings"
)

type Location struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

type House struct {
	ID          string   `gorm:"primaryKey"                   json:"id"`
	Price       int      `                                    json:"price"`
	PriceBucket string   `                                    json:"price_bucket"`
	Location    Location `gorm:"embedded;embeddedPrefix:loc_" json:"location"`
	Topology    string   `                                    json:"topology"`
	Description string   `                                    json:"description"`
}

type User struct {
	ID       string `gorm:"primaryKey"  json:"id"`
	Username string `gorm:"uniqueIndex" json:"username"`
}

type Filter struct {
	ID        string   `gorm:"primaryKey"                       json:"id"`
	QueueName string   `gorm:"uniqueIndex:idx_filter_user_queue" json:"queue_name"`
	UserID    string   `gorm:"uniqueIndex:idx_filter_user_queue" json:"user_id"`
	User      User     `gorm:"foreignKey:UserID"                json:"user"`
	Location  Location `gorm:"embedded;embeddedPrefix:loc_"      json:"location"`
	Topology  string   `                                         json:"topology"`
	MinPrice  int      `                                         json:"min_price"`
	MaxPrice  int      `                                         json:"max_price"`
}

// FilterQueueName derives a deterministic RabbitMQ queue name from filter
// fields. Both the API and the worker call this so they always agree on the
// same name without any coordination.
func FilterQueueName(country, region, city, topology string, minPrice, maxPrice int) string {
	priceRange := strconv.Itoa(minPrice) + "_" + strconv.Itoa(maxPrice)
	parts := []string{country, region, city, topology, priceRange}
	for i, p := range parts {
		parts[i] = strings.ToLower(strings.ReplaceAll(p, " ", "_"))
	}
	return "filter." + strings.Join(parts, ".")
}

// BucketsInRange returns all 100k-wide price buckets that overlap the range
// [minPrice, maxPrice). Both API and worker use this to bind/subscribe.
func BucketsInRange(minPrice, maxPrice int) []string {
	const size = 100000
	first := (minPrice / size) * size
	var buckets []string
	for b := first; b < maxPrice; b += size {
		buckets = append(buckets, strconv.Itoa(b)+"_"+strconv.Itoa(b+size))
	}
	return buckets
}
