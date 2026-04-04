package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	models "example.com/fst_demo/db"

	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

const exchangeName = "listings"

// hub routes JSON messages to WebSocket clients by user ID.
type hub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]struct{} // userID → connections
}

func newHub() *hub {
	return &hub{clients: make(map[string]map[*websocket.Conn]struct{})}
}

func (h *hub) add(userID string, c *websocket.Conn) {
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]struct{})
	}
	h.clients[userID][c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) remove(userID string, c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients[userID], c)
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
	h.mu.Unlock()
	c.Close()
}

// notify sends data only to connections belonging to the given user IDs.
func (h *hub) notify(userIDs []string, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, uid := range userIDs {
		for c := range h.clients[uid] {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				delete(h.clients[uid], c)
				c.Close()
			}
		}
	}
}

// deduper prevents the same (house, user) pair from triggering multiple
// notifications when a user has overlapping filters.
type deduper struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDeduper() *deduper {
	d := &deduper{seen: make(map[string]struct{})}
	go func() {
		for range time.NewTicker(30 * time.Minute).C {
			d.mu.Lock()
			d.seen = make(map[string]struct{})
			d.mu.Unlock()
		}
	}()
	return d
}

// firstSeen returns true the first time key is seen, false on subsequent calls.
func (d *deduper) firstSeen(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[key]; ok {
		return false
	}
	d.seen[key] = struct{}{}
	return true
}

type Notification struct {
	Queue string         `json:"queue"`
	House map[string]any `json:"house"`
	Users []models.User  `json:"users"`
}

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqps://ehuqhykx:OtTa4P11g-BT1iJDCfiyiSg6jFPe6DXG@dragonfly.rmq4.cloudamqp.com/ehuqhykx"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "../fst_demo.sqlite"
	}
	wsAddr := os.Getenv("WS_ADDR")
	if wsAddr == "" {
		wsAddr = ":8081"
	}

	database, err := models.Open(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("dial rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchangeName, "headers", true, false, false, false, nil); err != nil {
		log.Fatalf("declare exchange: %v", err)
	}

	h := newHub()
	dedup := newDeduper()
	startWSServer(h, wsAddr)

	var mu sync.Mutex
	subscribed := make(map[string]bool)

	subscribe := func(f models.Filter) {
		mu.Lock()
		defer mu.Unlock()
		if subscribed[f.QueueName] {
			return
		}
		msgs, err := subscribeQueue(ch, f)
		if err != nil {
			log.Printf("subscribe queue %q: %v", f.QueueName, err)
			return
		}
		subscribed[f.QueueName] = true
		log.Printf("subscribed: queue=%q (%d bucket(s))", f.QueueName, len(models.BucketsInRange(f.MinPrice, f.MaxPrice)))
		go consumeQueue(database, f.QueueName, msgs, h, dedup)
	}

	loadAndSubscribe := func() {
		var filters []models.Filter
		if err := database.Find(&filters).Error; err != nil {
			log.Printf("poll filters: %v", err)
			return
		}
		for _, f := range filters {
			subscribe(f)
		}
	}

	loadAndSubscribe()

	mu.Lock()
	log.Printf("worker ready — %d queue(s), ws on %s", len(subscribed), wsAddr)
	mu.Unlock()

	go func() {
		for range time.NewTicker(5 * time.Second).C {
			loadAndSubscribe()
		}
	}()

	chanErr := make(chan *amqp.Error, 1)
	ch.NotifyClose(chanErr)
	if err := <-chanErr; err != nil {
		log.Fatalf("amqp channel error: %v", err)
	}
}

func startWSServer(h *hub, addr string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id query param required", http.StatusBadRequest)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade: %v", err)
			return
		}
		h.add(userID, c)
		log.Printf("ws client connected: user=%s (%s)", userID, r.RemoteAddr)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				h.remove(userID, c)
				log.Printf("ws client disconnected: user=%s (%s)", userID, r.RemoteAddr)
				return
			}
		}
	})
	go func() {
		log.Printf("ws listening on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("ws server: %v", err)
		}
	}()
}

func subscribeQueue(ch *amqp.Channel, f models.Filter) (<-chan amqp.Delivery, error) {
	if _, err := ch.QueueDeclare(f.QueueName, true, false, false, false, nil); err != nil {
		return nil, err
	}
	for _, bucket := range models.BucketsInRange(f.MinPrice, f.MaxPrice) {
		bindArgs := amqp.Table{
			"x-match":      "all",
			"country":      strings.ToLower(f.Location.Country),
			"region":       strings.ToLower(f.Location.Region),
			"city":         strings.ToLower(f.Location.City),
			"topology":     strings.ToLower(f.Topology),
			"price_bucket": bucket,
		}
		if err := ch.QueueBind(f.QueueName, "", exchangeName, false, bindArgs); err != nil {
			return nil, err
		}
	}
	return ch.Consume(f.QueueName, "", true, false, false, false, nil)
}

func consumeQueue(db *gorm.DB, queueName string, msgs <-chan amqp.Delivery, h *hub, dedup *deduper) {
	for msg := range msgs {
		logMessage(db, h, dedup, queueName, msg)
	}
	log.Printf("[%s] delivery channel closed", queueName)
}

func logMessage(db *gorm.DB, h *hub, dedup *deduper, queue string, msg amqp.Delivery) {
	var house map[string]any
	if err := json.Unmarshal(msg.Body, &house); err != nil {
		log.Printf("[%s] received (unparseable body): %s", queue, msg.Body)
		return
	}
	out, _ := json.MarshalIndent(house, "", "  ")
	log.Printf("[%s] received\n  headers: %v\n  body: %s", queue, msg.Headers, out)

	var users []models.User
	if err := db.Joins("JOIN filters ON filters.user_id = users.id").
		Where("filters.queue_name = ?", queue).
		Find(&users).Error; err != nil {
		log.Printf("[%s] failed to look up subscribers: %v", queue, err)
		return
	}

	houseID, _ := house["id"].(string)
	var toNotifyIDs []string
	var toNotifyUsers []models.User
	for _, u := range users {
		if dedup.firstSeen(houseID + ":" + u.ID) {
			toNotifyIDs = append(toNotifyIDs, u.ID)
			toNotifyUsers = append(toNotifyUsers, u)
		}
	}

	if len(toNotifyIDs) == 0 {
		log.Printf("[%s] no new subscribers to notify (all already notified or none)", queue)
		return
	}
	log.Printf("[%s] notifying %d subscriber(s):", queue, len(toNotifyIDs))
	for _, u := range toNotifyUsers {
		log.Printf("  - %s (id=%s)", u.Username, u.ID)
	}

	data, _ := json.Marshal(Notification{Queue: queue, House: house, Users: toNotifyUsers})
	h.notify(toNotifyIDs, data)
}
