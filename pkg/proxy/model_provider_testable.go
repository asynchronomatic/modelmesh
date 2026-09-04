package proxy

import (
	"modelmesh/pkg/proxy/modeldex"
)

type TestableModelProvider struct {
}

func (t *TestableModelProvider) GetModels() []modeldex.ModelRoute {
	return nil
}
