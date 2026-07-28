package service

import (
	"errors"
	"time"

	"payment-service/internal/domain"
	"payment-service/internal/repository/inmemory"
)

type PaymentService struct {
	repo *inmemory.PaymentRepository
}

func NewPaymentService(repo *inmemory.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ProcessPayment(orderID string, amount float64, customer string) (domain.Payment, error) {
	if orderID == "" || amount <= 0 {
		return domain.Payment{}, errors.New("invalid payment data")
	}
	payment := domain.Payment{
		ID:        "pay-" + time.Now().Format("20060102150405"),
		OrderID:   orderID,
		Amount:    amount,
		Customer:  customer,
		Status:    domain.PaymentStatusSuccess,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Save(payment); err != nil {
		return domain.Payment{}, err
	}
	return payment, nil
}
