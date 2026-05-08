package app

import (
	"context"

	"eats/backend/common"
)

type CourierRepository interface {
	RegisterCourier(ctx context.Context, courier Courier) (CourierUUID, error)
}

type CourierUUID struct {
	common.UUID
}

type Courier struct {
	CourierUUID CourierUUID
	Name        string
	PhoneNumber string
	City        string
}

func (s *Service) RegisterCourier(ctx context.Context, courier Courier) (CourierUUID, error) {
	return s.courierRepository.RegisterCourier(ctx, courier)
}
