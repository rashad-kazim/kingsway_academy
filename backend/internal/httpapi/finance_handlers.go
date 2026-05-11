package httpapi

import (
	"context"
	"net/http"
	"strings"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/finance"
)

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.CreatePaymentInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		payment, err := s.finance.CreatePayment(ctx, principal, input)
		if err == nil {
			setAuditEntity(ctx, "payments", payment.ID, payment.BranchID)
		}
		return http.StatusCreated, payment, err
	})
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.PaymentStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	payments, total, err := s.finance.ListPaymentsPage(r.Context(), principal, r.URL.Query().Get("branch_id"), status, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, payments)
}

func (s *Server) paymentAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/payments/"), "/")
	if id == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	payment, err := s.finance.GetPayment(r.Context(), principal, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payment)
}

func (s *Server) createSalaryModel(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.CreateSalaryModelInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		model, err := s.finance.CreateSalaryModel(ctx, principal, input)
		if err == nil {
			setAuditEntity(ctx, "salary-models", model.ID, model.BranchID)
		}
		return http.StatusCreated, model, err
	})
}

func (s *Server) listSalaryModels(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	models, total, err := s.finance.ListSalaryModelsPage(r.Context(), principal, r.URL.Query().Get("branch_id"), page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, models)
}

func (s *Server) calculateSwapAllocation(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.SwapAllocationInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.finance.CalculateSwapAllocation(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
