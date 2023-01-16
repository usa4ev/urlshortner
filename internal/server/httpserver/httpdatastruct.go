package httpserver

// urlreq & urlres are, respectively, request and response structures
// used to decode and encode messages when dealing with JSON content-type.
type (
	urlreq struct {
		URL string `json:"url"`
	}
	urlres struct {
		Result string `json:"result"`
	}
)

// urlwid & urlwidres are, respectively, request and response structures
// bounded with correlation_id field.
type (
	urlwid struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}
	urlwidres struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
)

type statsData struct {
	Urls  int `json:"urls"`
	Users int `json:"users"`
}