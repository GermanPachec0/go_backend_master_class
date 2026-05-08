package app

import (
	"context"
)

type Review struct {
	CustomerName   string
	RestaurantName string
	CourierName    string
	Rating         int
	Comment        string
}

func (s *Service) MakeReview(
	ctx context.Context,
	review Review,
) error {
	return nil
}
