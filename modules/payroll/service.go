package payroll

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/finance"
	"github.com/Orazgeldiyew/hezzet_market_backend/modules/workerfinance"
	apperr "github.com/Orazgeldiyew/hezzet_market_backend/pkg/errors"
	"github.com/xuri/excelize/v2"
)

var periodRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type Service struct {
	repo     *Repository
	wfRepo   *workerfinance.Repository
	finRepo  *finance.Repository
}

func NewService(repo *Repository, wfRepo *workerfinance.Repository, finRepo *finance.Repository) *Service {
	return &Service{repo: repo, wfRepo: wfRepo, finRepo: finRepo}
}

// ── calculate ───────────────────────────────────────────────────────────────

func (s *Service) Calculate(ctx context.Context, req CalculateRequest, userID int64) (PayrollRun, error) {
	if !periodRe.MatchString(req.Period) {
		return PayrollRun{}, apperr.Validation("period must be YYYY-MM format")
	}

	// Get compensation
	comp, err := s.wfRepo.GetCompensation(ctx, req.WorkerID)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("COMPENSATION_NOT_FOUND", "compensation not configured for this worker")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	// Sum open fines
	finesCents, err := s.wfRepo.SumOpenFines(ctx, req.WorkerID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Sum open debts
	debtsCents, err := s.wfRepo.SumOpenDebts(ctx, req.WorkerID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Calculate net salary
	netSalary := comp.BaseSalaryCents - finesCents - debtsCents
	if netSalary < 0 {
		netSalary = 0
	}

	run := PayrollRun{
		WorkerID:        req.WorkerID,
		Period:          req.Period,
		BaseSalaryCents: comp.BaseSalaryCents,
		FinesCents:      finesCents,
		DebtsCents:      debtsCents,
		NetSalaryCents:  netSalary,
	}

	if err := s.repo.Create(ctx, &run); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	return run, nil
}

// ── pay ─────────────────────────────────────────────────────────────────────

func (s *Service) Pay(ctx context.Context, payrollID int64, paymentTypeCode string, userID int64) (PayrollRun, error) {
	// Resolve payment type
	ptID, err := s.finRepo.GetPaymentTypeIDByCode(ctx, paymentTypeCode)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.Validation("invalid payment_type_code")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}
	defer tx.Rollback(ctx)

	// Lock payroll run
	run, err := s.repo.LockPayrollRun(ctx, tx, payrollID)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("PAYROLL_NOT_FOUND", "payroll run not found")
		}
		return PayrollRun{}, apperr.Internal(err)
	}

	if run.Status != "calculated" {
		return PayrollRun{}, apperr.Validation(
			fmt.Sprintf("payroll run is '%s', expected 'calculated'", run.Status),
		)
	}

	uid := &userID

	// Create expense transaction
	reason := fmt.Sprintf("Salary for %s", run.Period)
	finTxn := finance.Transaction{
		Type:          "expense",
		AmountCents:   run.NetSalaryCents,
		RelatedTable:  "payroll",
		RelatedID:     &run.ID,
		Status:        "paid",
		Reason:        &reason,
		CreatedBy:     uid,
		PaymentTypeID: &ptID,
	}
	if err := s.finRepo.CreateTransaction(ctx, tx, &finTxn); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Create payment record (full amount)
	if run.NetSalaryCents > 0 {
		finPay := finance.Payment{
			TransactionID: finTxn.ID,
			PaymentTypeID: ptID,
			AmountCents:   run.NetSalaryCents,
			CreatedBy:     uid,
		}
		if err := s.finRepo.CreatePayment(ctx, tx, &finPay); err != nil {
			return PayrollRun{}, apperr.Internal(err)
		}
	}

	// Update payroll run
	run, err = s.repo.UpdateStatusPaid(ctx, tx, payrollID, finTxn.ID)
	if err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Mark fines as deducted
	if err := s.wfRepo.MarkFinesDeducted(ctx, tx, run.WorkerID); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	// Settle debts
	if err := s.wfRepo.SettleDebts(ctx, tx, run.WorkerID); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return PayrollRun{}, apperr.Internal(err)
	}

	return run, nil
}

// ── list ────────────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, workerID *int64, period *string, limit, offset int) (PayrollListResult, error) {
	items, total, err := s.repo.List(ctx, workerID, period, limit, offset)
	if err != nil {
		return PayrollListResult{}, apperr.Internal(err)
	}
	if items == nil {
		items = []PayrollRun{}
	}
	return PayrollListResult{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (PayrollRun, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return PayrollRun{}, apperr.NotFound("PAYROLL_NOT_FOUND", "payroll run not found")
		}
		return PayrollRun{}, apperr.Internal(err)
	}
	return run, nil
}

// ── export ───────────────────────────────────────────────────────────────────

// ExportPeriod builds an xlsx file in memory for all payroll runs in the given period.
func (s *Service) ExportPeriod(ctx context.Context, period string) ([]byte, error) {
	if !periodRe.MatchString(period) {
		return nil, apperr.Validation("period must be YYYY-MM format")
	}

	rows, err := s.repo.ListForExport(ctx, period)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("list for export: %w", err))
	}

	f := excelize.NewFile()
	sheet := "Payroll " + period
	f.SetSheetName("Sheet1", sheet)

	// ── Helpers ──────────────────────────────────────────────────────────────

	thinBorder := []excelize.Border{
		{Type: "left", Color: "D9D9D9", Style: 1},
		{Type: "right", Color: "D9D9D9", Style: 1},
		{Type: "top", Color: "D9D9D9", Style: 1},
		{Type: "bottom", Color: "D9D9D9", Style: 1},
	}
	tmtFmt := `#,##0.00" TMT"`

	// ── Styles ───────────────────────────────────────────────────────────────

	// Header: dark slate bg, white bold Arial 12, centered
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: "Arial", Size: 12, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F2937"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder,
	})

	// Text style — odd rows (white) and even rows (zebra gray)
	textStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 15},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})
	textZebraStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 15},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})

	// Money style (TMT format) — odd and even rows
	moneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 15},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})
	moneyZebraStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 15},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"F2F2F2"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})

	// Paid: green — money (Net Salary col) and text (Status col)
	paidMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 15, Bold: true, Color: "375623"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})
	paidTextStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 15, Bold: true, Color: "375623"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})

	// Calculated: orange — money (Net Salary col) and text (Status col)
	calcMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Calibri", Size: 15, Bold: true, Color: "833C00"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"FCE4D6"}, Pattern: 1},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       thinBorder,
		CustomNumFmt: &tmtFmt,
	})
	calcTextStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 15, Bold: true, Color: "833C00"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FCE4D6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    thinBorder,
	})

	// ── Header row ───────────────────────────────────────────────────────────
	headers := []string{"#", "Worker", "Position", "Period", "Base Salary", "Fines", "Debts", "Net Salary", "Status"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, 1, 22)

	// ── Column widths ────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 24)
	f.SetColWidth(sheet, "C", "C", 16)
	f.SetColWidth(sheet, "D", "D", 10)
	f.SetColWidth(sheet, "E", "H", 16)
	f.SetColWidth(sheet, "I", "I", 13)

	// ── Freeze header + autofilter ────────────────────────────────────────────
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
	f.AutoFilter(sheet, "A1:I1", nil)

	// ── Data rows ────────────────────────────────────────────────────────────
	for i, row := range rows {
		r := i + 2
		zebra := i%2 == 1

		// Base styles for this row
		ts := textStyle
		ms := moneyStyle
		if zebra {
			ts = textZebraStyle
			ms = moneyZebraStyle
		}

		// Net salary + status styles based on payment status
		netMS := calcMoneyStyle
		statusTS := calcTextStyle
		if row.Status == "paid" {
			netMS = paidMoneyStyle
			statusTS = paidTextStyle
		}

		f.SetCellValue(sheet, mustCell(1, r), i+1)
		f.SetCellValue(sheet, mustCell(2, r), row.WorkerName)
		f.SetCellValue(sheet, mustCell(3, r), row.Position)
		f.SetCellValue(sheet, mustCell(4, r), row.Period)
		f.SetCellValue(sheet, mustCell(5, r), centsToFloat(row.BaseSalaryCents))
		f.SetCellValue(sheet, mustCell(6, r), centsToFloat(row.FinesCents))
		f.SetCellValue(sheet, mustCell(7, r), centsToFloat(row.DebtsCents))
		f.SetCellValue(sheet, mustCell(8, r), centsToFloat(row.NetSalaryCents))
		f.SetCellValue(sheet, mustCell(9, r), row.Status)

		// Text cols: A–D
		f.SetCellStyle(sheet, mustCell(1, r), mustCell(4, r), ts)
		// Money cols: E–G (base salary, fines, debts)
		f.SetCellStyle(sheet, mustCell(5, r), mustCell(7, r), ms)
		// Net Salary (H) — money style, status-colored
		f.SetCellStyle(sheet, mustCell(8, r), mustCell(8, r), netMS)
		// Status (I) — text style, status-colored
		f.SetCellStyle(sheet, mustCell(9, r), mustCell(9, r), statusTS)

		f.SetRowHeight(sheet, r, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, apperr.Internal(fmt.Errorf("write xlsx: %w", err))
	}
	return buf.Bytes(), nil
}

func mustCell(col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	return cell
}

func centsToFloat(cents int64) float64 {
	return float64(cents) / 100.0
}
