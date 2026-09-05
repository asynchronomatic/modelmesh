package proxy

import (
	"github.com/asynchronomatic/speakeasy/pkg/proxy/modeldex"
)

type TestableModelProvider struct {
}

func (t *TestableModelProvider) GetModels() []modeldex.ModelRoute {
	return nil
}
