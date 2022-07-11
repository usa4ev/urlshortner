package app

var urlMap map[int]string
var i int

func ShortURL(URL string) int {

	if urlMap == nil {
		urlMap = make(map[int]string)
	}

	for i, v := range urlMap {
		if v == URL {
			return i
		}
	}

	i++
	urlMap[i] = URL
	return i
}

func GetPath(id int) string {
	return urlMap[id]
}
