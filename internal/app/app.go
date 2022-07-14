package app

var (
	urlMap map[int]string
	i      int
)

func ShortURL(url string) int {
	if urlMap == nil {
		urlMap = make(map[int]string)
	}

	for i, v := range urlMap {
		if v == url {
			return i
		}
	}

	i++

	urlMap[i] = url

	return i
}

func GetPath(id int) string {
	return urlMap[id]
}
