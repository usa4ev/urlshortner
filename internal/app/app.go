package app

type (
	urlMap  map[int]string
	Storage struct {
		urlMap urlMap
		id     int
	}
)

func NewStorage() *Storage {
	return &Storage{make(urlMap), 0}
}

func ShortURL(url string, storage *Storage) int {
	for id, v := range storage.urlMap {
		if v == url {
			return id
		}
	}

	storage.id++

	storage.urlMap[storage.id] = url

	return storage.id
}

func GetPath(id int, storage *Storage) string {
	return storage.urlMap[id]
}
