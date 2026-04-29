package domain

import "time"

type FileCategory string

const (
	FileCategoryStandard FileCategory = "standard"
	FileCategorySpecial  FileCategory = "special"
)

type FilePurpose string

const (
	FilePurposeMaterial    FilePurpose = "material"
	FilePurposePayment     FilePurpose = "payment_receipt"
	FilePurposeEvent       FilePurpose = "event"
	FilePurposePassport    FilePurpose = "passport"
	FilePurposeDiploma     FilePurpose = "diploma"
	FilePurposeProfile     FilePurpose = "profile_photo"
	FilePurposeExamWriting FilePurpose = "exam_writing"
)

const WritingRetention = 60 * 24 * time.Hour

func (c FileCategory) IsValid() bool {
	switch c {
	case FileCategoryStandard, FileCategorySpecial:
		return true
	default:
		return false
	}
}

func (c FileCategory) CanOptimize() bool {
	return c == FileCategoryStandard
}

func (c FileCategory) MustPreserveOriginalBytes() bool {
	return c == FileCategorySpecial
}

func RetentionUntil(purpose FilePurpose, uploadedAt time.Time) *time.Time {
	if purpose != FilePurposeExamWriting {
		return nil
	}

	retentionUntil := uploadedAt.Add(WritingRetention)
	return &retentionUntil
}
