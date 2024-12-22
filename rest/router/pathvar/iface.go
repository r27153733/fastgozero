package pathvar

type Params interface {
	Get(string) (string, bool)
	VisitAll(f func(key, value string))
	Value(key string) (any, bool)
	Len() int
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

func (m MapParams) Value(key string) (any, bool) {
	v, ok := m[key]
	return v, ok
}

func (m MapParams) Len() int {
	return len(m)
}
