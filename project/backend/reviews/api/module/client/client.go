package client

import (
	"context"
)

type Review interface {
	MakeReview(ctx context.Context, req MakeReviewRequest) (MakeReviewResponse, error)
}

type MakeReviewRequest struct {
	CustomerName   string
	RestaurantName string
	CourierName    string
	Rating         int
	Comment        string
}

type MakeReviewResponse struct {
}
