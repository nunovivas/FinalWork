package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	models "example.com/fst_demo/db"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

//go:embed index.html
var indexHTML []byte

type App struct {
	db     *gorm.DB
	amqpCh *amqp.Channel
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

	database, err := models.Open(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	log.Printf("db: opened %s", dbPath)

	_, amqpCh, err := connectRabbitMQ(rabbitURL)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}

	app := &App{
		db:     database,
		amqpCh: amqpCh,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/login", app.handleLogin)
	mux.HandleFunc("/users", app.handleUsers)
	mux.HandleFunc("/houses", app.handleHouses)
	mux.HandleFunc("/houses/", app.handleHouseByID)
	mux.HandleFunc("/filters", app.handleFilters)
	mux.HandleFunc("/create-house", app.handleCreateHouse)
	mux.HandleFunc("/create-filter", app.handleCreateFilter)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	var user models.User
	err := a.db.Where("username = ?", req.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = models.User{ID: newID(), Username: req.Username}
		if err := a.db.Create(&user).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create user")
			return
		}
		log.Printf("user created: id=%s username=%s", user.ID, user.Username)
		writeJSON(w, http.StatusCreated, user)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to look up user")
		return
	}

	log.Printf("user logged in: id=%s username=%s", user.ID, user.Username)
	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	var users []models.User
	if err := a.db.Find(&users).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (a *App) handleHouses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	var houses []models.House
	if err := a.db.Find(&houses).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch houses")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"houses": houses})
}

func (a *App) handleHouseByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w, http.MethodDelete)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/houses/")
	if id == "" || id == r.URL.Path {
		writeError(w, http.StatusBadRequest, "missing house id")
		return
	}

	result := a.db.Delete(&models.House{}, "id = ?", id)
	if result.Error != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete house")
		return
	}
	if result.RowsAffected == 0 {
		writeError(w, http.StatusNotFound, "house not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "house removed", "id": id})
}

func (a *App) handleFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	var filters []models.Filter
	if err := a.db.Preload("User").Find(&filters).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch filters")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"filters": filters})
}

func (a *App) handleCreateHouse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req CreateHouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	house, err := buildHouse(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.db.Create(&house).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save house")
		return
	}

	log.Printf("house created: id=%s city=%s price_bucket=%s", house.ID, house.Location.City, house.PriceBucket)

	if err := publishHouseEvent(a.amqpCh, house); err != nil {
		log.Printf("rabbitmq publish error: %v", err)
	}

	writeJSON(w, http.StatusCreated, house)
}

func (a *App) handleCreateFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req CreateFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	var user models.User
	if err := a.db.First(&user, "id = ?", req.UserID).Error; err != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}

	filter, err := buildFilter(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := a.db.Create(&filter).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save filter")
		return
	}

	log.Printf("filter created: id=%s queue=%s", filter.ID, filter.QueueName)

	if err := bindFilterQueue(a.amqpCh, filter); err != nil {
		log.Printf("rabbitmq bind error: %v", err)
	}

	writeJSON(w, http.StatusCreated, filter)
}

func buildHouse(req CreateHouseRequest) (models.House, error) {
	if req.Price <= 0 {
		return models.House{}, errors.New("price must be greater than 0")
	}
	if err := validateLocation(req.Location); err != nil {
		return models.House{}, err
	}
	if strings.TrimSpace(req.Topology) == "" {
		return models.House{}, errors.New("topology is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return models.House{}, errors.New("description is required")
	}

	return models.House{
		ID:          newID(),
		Price:       req.Price,
		PriceBucket: priceBucketFor(req.Price),
		Location:    normalizeLocation(req.Location),
		Topology:    strings.TrimSpace(req.Topology),
		Description: strings.TrimSpace(req.Description),
	}, nil
}

func buildFilter(req CreateFilterRequest) (models.Filter, error) {
	if err := validateLocation(req.Location); err != nil {
		return models.Filter{}, err
	}
	if strings.TrimSpace(req.Topology) == "" {
		return models.Filter{}, errors.New("topology is required")
	}

	var minPrice, maxPrice int

	if pb := strings.TrimSpace(req.PriceBucket); pb != "" {
		parts := strings.SplitN(pb, "_", 2)
		if len(parts) != 2 {
			return models.Filter{}, errors.New("invalid price_bucket, expected format: 100000_200000")
		}
		var err error
		if minPrice, err = strconv.Atoi(parts[0]); err != nil || minPrice < 0 {
			return models.Filter{}, errors.New("invalid price_bucket lower bound")
		}
		if maxPrice, err = strconv.Atoi(parts[1]); err != nil || maxPrice <= minPrice {
			return models.Filter{}, errors.New("invalid price_bucket upper bound")
		}
	} else {
		if req.MinPrice < 0 || req.MaxPrice < 0 {
			return models.Filter{}, errors.New("price values cannot be negative")
		}
		if req.MinPrice == 0 && req.MaxPrice == 0 {
			return models.Filter{}, errors.New("either price_bucket or min/max price must be provided")
		}
		if req.MaxPrice > 0 && req.MinPrice >= req.MaxPrice {
			return models.Filter{}, errors.New("min_price must be less than max_price")
		}
		minPrice = req.MinPrice
		maxPrice = req.MaxPrice
	}

	loc := normalizeLocation(req.Location)
	topology := strings.TrimSpace(req.Topology)

	return models.Filter{
		ID:        newID(),
		QueueName: models.FilterQueueName(loc.Country, loc.Region, loc.City, topology, minPrice, maxPrice),
		UserID:    req.UserID,
		Location:  loc,
		Topology:  topology,
		MinPrice:  minPrice,
		MaxPrice:  maxPrice,
	}, nil
}

func validateLocation(loc models.Location) error {
	if strings.TrimSpace(loc.Country) == "" {
		return errors.New("location.country is required")
	}
	if strings.TrimSpace(loc.Region) == "" {
		return errors.New("location.region is required")
	}
	if strings.TrimSpace(loc.City) == "" {
		return errors.New("location.city is required")
	}
	return nil
}

func normalizeLocation(loc models.Location) models.Location {
	return models.Location{
		City:    strings.TrimSpace(loc.City),
		Region:  strings.TrimSpace(loc.Region),
		Country: strings.TrimSpace(loc.Country),
	}
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "fallback-id"
	}
	return hex.EncodeToString(b[:])
}

func priceBucketFor(price int) string {
	if price < 0 {
		price = 0
	}
	size := 100000
	lower := (price / size) * size
	upper := lower + size
	return strconv.Itoa(lower) + "_" + strconv.Itoa(upper)
}


func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	if len(allowedMethods) > 0 {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
