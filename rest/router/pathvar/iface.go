package pathvar

type Params interface {
	Get(string) (string, bool)
	VisitAll(f func(key, value string))
	Len() int
}

type EmptyParams struct{}

func (m EmptyParams) Get(key string) (string, bool) {
	return "", false
}

func (m EmptyParams) VisitAll(_ func(key, value string)) {
	return
}

func (m EmptyParams) Len() int {
	return 0
}

type MapParams map[string]string

func (m MapParams) Get(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

func (m MapParams) VisitAll(f func(key, value string)) {
	for k, v := range m {
		f(k, v)
	}
}

func (m MapParams) Len() int {
	return len(m)
}
