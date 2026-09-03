package proxy

type TestableModelProvider struct {
}

func (t *TestableModelProvider) GetModels() []*ModelRoute {
	return nil
}
