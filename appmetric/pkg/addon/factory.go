package addon

import (
	"fmt"
	"github.com/turbonomic/prometurbo/appmetric/pkg/alligator"
)

const (
	RedisGetterCategory     = "Redis"
	IstioGetterCategory     = "Istio"
	IstioVAppGetterCategory = "Istio.VApp"
)

type GetterFactory struct {
}

func NewGetterFactory() *GetterFactory {
	return &GetterFactory{}
}

func (f *GetterFactory) CreateEntityGetter(category, name string) (alligator.EntityMetricGetter, error) {
	switch category {
	case RedisGetterCategory:
		return NewRedisEntityGetter(name), nil
	case IstioGetterCategory:
		g := newIstioEntityGetter(name)
		forVapp := false
		g.SetType(forVapp)
		return g, nil
	case IstioVAppGetterCategory:
		g := newIstioEntityGetter(name)
		forVapp := true
		g.SetType(forVapp)
		return g, nil
	}

	return nil, fmt.Errorf("Unknown category: %v", category)
}
