package reviews

import (
	"context"

	"eats/backend/common"
	"eats/backend/common/module"
	"eats/backend/common/module/contracts"
	client "eats/backend/reviews/api/module"
	"eats/backend/reviews/app"
)

type Module struct {
	service *app.Service
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) Name() module.Name {
	return "reviews"
}

func (m *Module) Init(ctx context.Context) error {
	m.service = app.NewService()
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, contracts *contracts.Contracts) error {
	contracts.Review = client.New(m.service)
	return nil
}

func (m *Module) RegisterHttp(ctx context.Context, e common.EchoRouter) error {
	// this module doesn't expose any HTTP endpoints
	return nil
}
