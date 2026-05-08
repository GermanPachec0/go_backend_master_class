package module

import (
	"context"
	"fmt"

	"eats/backend/reviews/api/module/client"
	"eats/backend/reviews/app"
)

type Review struct {
	service *app.Service
}

func New(service *app.Service) *Review {
	if service == nil {
		panic("service cannot be nil")
	}

	return &Review{service: service}
}

func (i Review) MakeReview(ctx context.Context, req client.MakeReviewRequest) (client.MakeReviewResponse, error) {
	err := i.service.MakeReview(
		ctx,
		req.Review,
	)
	if err != nil {
		return client.MakeReviewResponse{}, fmt.Errorf("failed to make review: %w", err)
	}

	return client.MakeReviewResponse{}, nil
}
