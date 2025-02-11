package api

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

type Config interface {
	DurationInterval() (int, int)
	SetDurationInterval(min, max int) error
	ErrorsPercentage() float64
	SetErrorsPercentage(value float64) error
	RequestsHour() int
	SetRequestsHour(reqHour int) error
}

type Handler struct {
	Config  Config
	Metrics http.Handler

	once    sync.Once
	handler http.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(h.setupHandlers)
	h.handler.ServeHTTP(w, r)
}

func (h *Handler) setupHandlers() {
	router := mux.NewRouter()

	h.setupHealthHandler(router)
	h.setupDurationIntervalHandlers(router)
	h.setupErrorsPercentageHandlers(router)
	h.setupRequestsHourHandlers(router)
	h.setupMetricsHandler(router)
	h.setupRootHandler(router)

	h.handler = router
}

func (h *Handler) setupRootHandler(router *mux.Router) {
	router.
		Methods(http.MethodGet).
		Path("/").
		HandlerFunc(h.handleRoot)
}

func (h *Handler) setupHealthHandler(router *mux.Router) {
	router.
		Methods(http.MethodGet).
		Path("/-/health").
		HandlerFunc(h.handleHealth)
}

func (h *Handler) setupDurationIntervalHandlers(router *mux.Router) {
	sub := router.
		PathPrefix("/-/config/duration-interval").
		Subrouter()

	sub.
		Methods(http.MethodGet).
		HandlerFunc(h.handleGetDurationInterval)

	sub.
		Methods(http.MethodPut).
		HandlerFunc(h.handleSetDurationInterval)
}

func (h *Handler) setupErrorsPercentageHandlers(router *mux.Router) {
	sub := router.
		PathPrefix("/-/config/errors-percentage").
		Subrouter()

	sub.
		Methods(http.MethodGet).
		HandlerFunc(h.handleGetErrorsPercentage)

	sub.
		Methods(http.MethodPut).
		HandlerFunc(h.handleSetErrorsPercentage)
}

func (h *Handler) setupRequestsHourHandlers(router *mux.Router) {
	sub := router.
		PathPrefix("/-/config/requests-hour").
		Subrouter()

	sub.
		Methods(http.MethodGet).
		HandlerFunc(h.handleGetRequestsHour)

	sub.
		Methods(http.MethodPut).
		HandlerFunc(h.handleSetRequestsHour)
}

func (h *Handler) setupMetricsHandler(router *mux.Router) {
	router.
		Methods(http.MethodGet).
		Path("/metrics").
		Handler(h.Metrics)
}

//go:embed files/index.html
var index string

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", " text/html")

	type Data struct {
		ErrorsPercentage    float64
		MinDurationInterval int
		MaxDurationInterval int
		ReqHour             int
	}

	minD, maxD := h.Config.DurationInterval()

	data := Data{
		ErrorsPercentage:    h.Config.ErrorsPercentage(),
		MinDurationInterval: minD,
		MaxDurationInterval: maxD,
		ReqHour:             h.Config.RequestsHour(),
	}

	tmpl, err := template.New("index").Parse(index)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "generating template: %v", err)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "executing template: %v", err)
		return
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func (h *Handler) handleGetDurationInterval(w http.ResponseWriter, r *http.Request) {
	min, max := h.Config.DurationInterval()
	fmt.Fprintf(w, "%d,%d\n", min, max)
}

func (h *Handler) handleSetDurationInterval(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "read body: %v", err)
		return
	}

	min, max, err := parseDurationInterval(string(data))
	if err != nil {
		httpError(w, http.StatusBadRequest, "parse duration interval: %v", err)
		return
	}

	if err := h.Config.SetDurationInterval(min, max); err != nil {
		httpError(w, http.StatusBadRequest, "set duration interval: %v", err)
		return
	}

	fmt.Fprintln(w, "OK")
}

func (h *Handler) handleGetErrorsPercentage(w http.ResponseWriter, r *http.Request) {
	s := strconv.FormatFloat(h.Config.ErrorsPercentage(), 'f', -1, 64)
	fmt.Fprintf(w, "%s\n", s)
}

func (h *Handler) handleSetErrorsPercentage(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "read body: %v", err)
		return
	}

	value, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		httpError(w, http.StatusBadRequest, "parse errors percentage: %v", err)
		return
	}

	if err := h.Config.SetErrorsPercentage(value); err != nil {
		httpError(w, http.StatusBadRequest, "set errors percentage: %v", err)
		return
	}

	fmt.Fprintln(w, "OK")
}

func httpError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	http.Error(w, fmt.Sprintf(format, args...), code)
}

func (h *Handler) handleGetRequestsHour(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%d\n", h.Config.RequestsHour())
}

func (h *Handler) handleSetRequestsHour(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "read body: %v", err)
		return
	}

	value, err := strconv.Atoi(string(data))
	if err != nil {
		httpError(w, http.StatusBadRequest, "parse requests hour: %v", err)
		return
	}

	if err := h.Config.SetRequestsHour(value); err != nil {
		httpError(w, http.StatusBadRequest, "set requests hour: %v", err)
		return
	}

	fmt.Fprintln(w, "OK")
}
