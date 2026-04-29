package domain

type SalaryModelType string

const (
	SalaryModelFixed   SalaryModelType = "fixed"
	SalaryModelPercent SalaryModelType = "percent"
	SalaryModelHybrid  SalaryModelType = "hybrid"
)

func (s SalaryModelType) IsValid() bool {
	switch s {
	case SalaryModelFixed, SalaryModelPercent, SalaryModelHybrid:
		return true
	default:
		return false
	}
}

func CalculateSwapAllocation(totalAmountCents int64, firstTeacherDays, secondTeacherDays int) (int64, int64) {
	totalDays := firstTeacherDays + secondTeacherDays
	if totalAmountCents <= 0 || totalDays <= 0 {
		return 0, 0
	}

	first := totalAmountCents * int64(firstTeacherDays) / int64(totalDays)
	return first, totalAmountCents - first
}
